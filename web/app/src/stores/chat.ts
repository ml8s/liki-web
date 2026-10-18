import { defineStore } from 'pinia'
import type { ChatMessage, ChatSession } from '@/api/chat'

// @ag-ui/client 较重（~292KB raw）：仅在真正发送消息时动态加载，
// 避免 dashboard 依赖链把 chat transport 拖入首屏。
type ChatApi = typeof import('@/api/chat')
let chatApi: ChatApi | null = null
async function chatApiLazy(): Promise<ChatApi> {
  if (!chatApi) chatApi = await import('@/api/chat')
  return chatApi
}
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { errorMessage } from '@/utils/errors'
import { chatSkills } from '@/config/chat'
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
        this.sessions = await (await chatApiLazy()).fetchChatSessions()
      } catch (error) {
        this.error = errorMessage(error)
      } finally {
        this.loadingSessions = false
      }
    },
    async createSession(profileId: string, expert: string) {
      const entitlements = useEntitlementsStore()
      await entitlements.load()
      if (!entitlements.hasSkill('liki')) {
        throw new Error(t('chat.skillNotIncluded'))
      }
      const nickname = useBirthProfilesStore().list.find(p => p.id === profileId)?.nickname ?? ''
      const label = skillLabel(expert)
      const title = nickname
        ? t('chat.sessionTitleWithBirthProfile', { nickname, skill: label })
        : t('chat.sessionTitle', { skill: label })
      const session = await (await chatApiLazy()).createChatSession(profileId, 'liki', expert, title)
      this.sessions.unshift(session)
      return session
    },
    /** 取该档案与专家的会话：有则复用（继续），无则新建。 */
    async ensureSession(profileId: string, expert: string) {
      const existing = this.sessions.find(s => s.profile_id === profileId && s.expert === expert)
      if (existing) return existing
      return this.createSession(profileId, expert)
    },
    async renameSession(sessionId: string, title: string) {
      const updated = await (await chatApiLazy()).renameChatSession(sessionId, title)
      const index = this.sessions.findIndex(s => s.id === sessionId)
      if (index >= 0) this.sessions[index] = updated
      if (this.activeSession?.id === sessionId) this.activeSession = updated
    },
    async removeSession(sessionId: string) {
      await (await chatApiLazy()).deleteChatSession(sessionId)
      this.sessions = this.sessions.filter(s => s.id !== sessionId)
      if (this.activeSession?.id === sessionId) this.activeSession = null
    },
    async openSession(sessionId: string) {
      // 切换会话即中止进行中的流：旧请求的完成回调不得写入新会话。
      this.stopGeneration()
      this.loadingMessages = true
      this.error = ''
      this.activeSession = this.sessions.find(item => item.id === sessionId) ?? null
      try {
        if (!this.activeSession) {
          this.activeSession = await (await chatApiLazy()).fetchChatSession(sessionId)
        }
        this.messages = await (await chatApiLazy()).fetchChatMessages(sessionId)
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
      // 先捕获 signal：stopGeneration 会把 streamController 置空，而参数在
      // await 动态导入之后才求值，直接读会踩 null（真实竞态）。
      const signal = streamController.signal

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
        const result = await (await chatApiLazy()).streamChatMessage(session.id, message, event => {
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
        }, signal)

        if (this.activeSession?.id !== session.id) {
          // 用户已切换会话：结果属旧会话，权威消息由服务端持久化，前端不写入。
          return
        }
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
        const aborted = (error as { name?: string } | null)?.name === 'AbortError'
        this.phase = ''
        streamController = null
        if (textTimer) { clearTimeout(textTimer); textTimer = null }
        assistant.content = pendingText
        assistant.thinking = pendingThinking
        if (aborted) {
          // 停止生成：保留本地已流出内容并标记「已停止」；部分答案不得静默消失。
          assistant.stopped = true
          return
        }
        this.error = errorMessage(error)
        try {
          this.messages = await (await chatApiLazy()).fetchChatMessages(session.id)
        } catch {
          // 保留本地消息，标记为失败状态
          assistant.error = this.error || t('chat.sendFailed')
        }
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
      this.stopGeneration()
      this.sessions = []
      this.resetActive()
      this.loadingSessions = false
      this.loadingMessages = false
      this.sending = false
      this.pendingIntent = ''
    },
  },
})

/** 会话展示名：专家标签（config/chat.ts 单源）；历史会话（无专家）回退到「命理」。 */
export function skillLabel(expert: string): string {
  const skill = chatSkills.find(item => item.id === expert)
  return i18n.global.t(skill?.labelKey ?? 'services.skill.compact')
}
