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
import { useProfilesStore } from '@/stores/profiles'
import { errorMessage } from '@/utils/errors'
import { i18n } from '@/i18n'

function t(key: string, values?: Record<string, unknown>): string {
  return i18n.global.t(key, values ?? {})
}

let streamController: AbortController | null = null

function localId(): string {
  return `local-${crypto.randomUUID()}-${Date.now()}`
}

interface ChatState {
  sessions: ChatSession[]
  activeSession: ChatSession | null
  messages: ChatMessage[]
  loadingSessions: boolean
  loadingMessages: boolean
  sending: boolean
  phase: string
  error: string
  /** 待自动发送的首条意图消息（服务入口 → 会话打开后由 ChatSessionView 发送并清空；内存态，刷新即失） */
  pendingIntent: string
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
    pendingIntent: '',
  }),
  actions: {
    async loadSessions(force = false) {
      if (this.loadingSessions || (this.sessions.length && !force)) return
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
        throw new Error(t('chat.skillNotIncluded'))
      }
      const nickname = useProfilesStore().list.find(p => p.id === profileId)?.nickname ?? ''
      const title = nickname
        ? t('chat.sessionTitleWithProfile', { nickname, skill: skillLabel(skill) })
        : t('chat.sessionTitle', { skill: skillLabel(skill) })
      const session = await createChatSession(profileId, skill, title)
      this.sessions.unshift(session)
      return session
    },
    /** 取该档案的会话：有则复用（继续），无则新建。 */
    async ensureSession(profileId: string) {
      const existing = this.sessions.find(s => s.profile_id === profileId)
      if (existing) return existing
      return this.createSession(profileId, 'liki')
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
      this.phase = t('chat.phase.thinking')
      streamController = new AbortController()

      const userId = localId()
      const assistantId = localId()
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
            this.phase = this.phase || t('chat.phase.thinking')
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
          // 保留本地消息，标记为失败状态
          assistant.error = this.error || t('chat.sendFailed')
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
      this.pendingIntent = ''
    },
  },
})

export function skillLabel(skill: string): string {
  // Session titles append “reading”; keep the skill label compact.
  return skill === 'liki' ? i18n.global.t('services.skill.compact') : skill
}
