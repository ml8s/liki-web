import { defineStore } from 'pinia'
import {
  createChatSession,
  fetchChatSession,
  fetchChatMessages,
  fetchChatSessions,
  streamChatMessage,
  type ChatMessage,
  type ChatSession,
} from '@/api/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { toUserMessage } from '@/utils/errors'

let streamController: AbortController | null = null

interface ChatState {
  sessions: ChatSession[]
  activeSession: ChatSession | null
  messages: ChatMessage[]
  loadingSessions: boolean
  loadingMessages: boolean
  sending: boolean
  phase: string
  error: string
}

export const useChatStore = defineStore('chat', {
  state: (): ChatState => ({
    sessions: [],
    activeSession: null,
    messages: [],
    loadingSessions: false,
    loadingMessages: false,
    sending: false,
    phase: '',
    error: '',
  }),
  getters: {
    canSend: state => !!state.activeSession && !state.sending,
  },
  actions: {
    async loadSessions() {
      this.loadingSessions = true
      this.error = ''
      try {
        this.sessions = await fetchChatSessions()
      } catch (error) {
        this.error = errorMessage(error)
      } finally {
        this.loadingSessions = false
      }
    },
    async createSession(profileId: string, skill: string) {
      const entitlements = useEntitlementsStore()
      await entitlements.load()
      if (!entitlements.hasSkill(skill)) {
        throw new Error('当前订单未开通该 skill')
      }
      const session = await createChatSession(profileId, skill, `${skillLabel(skill)}咨询`)
      this.sessions.unshift(session)
      return session
    },
    async openSession(sessionId: string) {
      this.loadingMessages = true
      this.error = ''
      this.activeSession = this.sessions.find(item => item.id === sessionId) ?? null
      try {
        if (!this.activeSession) {
          this.activeSession = await fetchChatSession(sessionId)
        }
        this.messages = await fetchChatMessages(sessionId)
      } catch (error) {
        this.messages = []
        this.activeSession = null
        this.error = errorMessage(error)
      } finally {
        this.loadingMessages = false
      }
    },
    async send(content: string) {
      const session = this.activeSession
      const message = content.trim()
      if (!session || !message || this.sending) return
      this.sending = true
      this.error = ''
      this.phase = '思考中'
      streamController = new AbortController()

      const userId = `local-user-${Date.now()}`
      const assistantId = `local-assistant-${Date.now()}`
      this.messages.push({
        id: userId,
        session_id: session.id,
        user_id: session.user_id,
        role: 'user',
        kind: 'text',
        content: message,
        created_at: new Date().toISOString(),
      })
      const assistant: ChatMessage = {
        id: assistantId,
        session_id: session.id,
        user_id: session.user_id,
        role: 'assistant',
        kind: 'text',
        content: '',
        created_at: new Date().toISOString(),
      }
      this.messages.push(assistant)

      // 流式渲染节流：累积 delta，≤120ms 刷新一次（借命名 chat 实践），
      // 避免高频 delta 触发响应式全量重渲染。thinking-delta 一并节流。
      let pendingText = ''
      let pendingThinking = ''
      let textTimer: ReturnType<typeof setTimeout> | null = null
      const flushText = () => {
        assistant.content = pendingText
        assistant.thinking = pendingThinking
        textTimer = null
      }

      try {
        const result = await streamChatMessage(session.id, message, event => {
          if (event.type === 'thinking') {
            this.phase = this.phase || '思考中'
            return
          }
          if (event.type === 'phase') {
            this.phase = event.content || this.phase
            return
          }
          if (event.type === 'thinking-delta') {
            pendingThinking += event.content ?? ''
            if (!textTimer) textTimer = setTimeout(flushText, 120)
            return
          }
          if (event.type === 'text-delta') {
            this.phase = ''
            pendingText += event.content ?? ''
            if (!textTimer) textTimer = setTimeout(flushText, 120)
          }
        }, streamController.signal)

        if (textTimer) { clearTimeout(textTimer); textTimer = null }
        assistant.content = pendingText
        assistant.thinking = pendingThinking
        this.messages = this.messages.filter(item => item.id !== userId && item.id !== assistantId)
        const finalAssistant = result.assistant_message
        if (pendingThinking) finalAssistant.thinking = pendingThinking
        this.messages.push(result.user_message, finalAssistant)
        const index = this.sessions.findIndex(item => item.id === session.id)
        if (index >= 0) {
          this.sessions[index] = result.session
        } else {
          this.sessions.unshift(result.session)
        }
        this.activeSession = result.session
        this.phase = ''
      } catch (error) {
        const aborted = error instanceof DOMException && error.name === 'AbortError'
        if (!aborted) this.error = errorMessage(error)
        this.phase = ''
        streamController = null
        if (textTimer) { clearTimeout(textTimer); textTimer = null }
        assistant.content = pendingText
        assistant.thinking = pendingThinking
        try {
          this.messages = await fetchChatMessages(session.id)
        } catch {
          this.messages = this.messages.filter(item => item.id !== userId && item.id !== assistantId)
        }
        if (aborted) return
      } finally {
        streamController = null
        this.sending = false
      }
    },
    stopGeneration() {
      streamController?.abort()
      streamController = null
    },
    resetActive() {
      this.activeSession = null
      this.messages = []
      this.error = ''
    },
    reset() {
      this.sessions = []
      this.resetActive()
      this.loadingSessions = false
      this.loadingMessages = false
      this.sending = false
    },
  },
})

export function skillLabel(skill: string): string {
  const labels: Record<string, string> = {
    'liki': '命理咨询',
    'liki-naming': '起名',
    'liki-bazi': '八字紫微',
    'liki-divination': '问卦',
    'liki-fengshui': '风水',
  }
  return labels[skill] ?? skill
}

export function errorMessage(error: unknown): string {
  return toUserMessage(error, '请求失败')
}
