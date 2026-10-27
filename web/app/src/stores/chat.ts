import { defineStore } from 'pinia'
import type { ChatMessage, ChatSession } from '@/api/chat'
import type { ConsultationContext } from '@shared/domain/consultation'

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
import { toolPhaseKey } from '@/utils/toolPhase'
import { scenarioById } from '@/config/chat'
import { i18n } from '@/i18n'
import { reportClientError } from '@shared/utils/client-error'

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
  /** 更早消息的分页游标（空 = 已到顶）。 */
  messagesCursor: string
  loadingEarlier: boolean
  loadingSessions: boolean
  loadingMessages: boolean
  sending: boolean
  phase: string
  error: string
  loadSessionsError: unknown
  /** 待自动发送的首条意图消息（服务入口 → 会话打开后由 ChatSessionView 发送并清空；内存态，刷新即失） */
  pendingIntent: string
}

export const useChatStore = defineStore('chat', {
  state: (): ChatState => ({
    sessions: [],
    activeSession: null,
    messages: [],
    messagesCursor: '',
    loadingEarlier: false,
    loadingSessions: false,
    loadingMessages: false,
    sending: false,
    phase: '',
    error: '',
    loadSessionsError: null,
    pendingIntent: '',
  }),
  actions: {
    async loadSessions(force = false) {
      if (this.loadingSessions || (this.sessions.length && !force)) return
      this.loadingSessions = true
      this.error = ''
      try {
        this.sessions = await (await chatApiLazy()).fetchChatSessions()
        this.loadSessionsError = null
      } catch (error) {
        this.loadSessionsError = error
        this.error = errorMessage(error)
      } finally {
        this.loadingSessions = false
      }
    },
    /** 按用户场景创建会话：标题用场景名（命理/流年/…），内部路由到场景对应专家。 */
    async createSession(profileId: string, scenarioId: string, context?: ConsultationContext) {
      const entitlements = useEntitlementsStore()
      await entitlements.load()
      if (!entitlements.hasSkill('liki')) {
        throw new Error(t('chat.skillNotIncluded'))
      }
      const scenario = scenarioById(scenarioId)
      const expert = scenario?.expert ?? 'natal'
      const nickname = useBirthProfilesStore().list.find(p => p.id === profileId)?.nickname ?? ''
      const label = scenario ? t(scenario.labelKey) : t('services.natal.label')
      const title = nickname
        ? t('chat.sessionTitleWithBirthProfile', { nickname, skill: label })
        : t('chat.sessionTitle', { skill: label })
      const session = await (await chatApiLazy()).createChatSession(
        profileId,
        'liki',
        expert,
        title,
        context,
        scenarioId,
      )
      this.sessions.unshift(session)
      // 返回数组内的响应式代理：返回 unshift 前的原始对象会让后续写入绕过
      // 响应式（与聊天气泡同类的历史 bug，见 stores/chat.ts send()）。
      return this.sessions[0]
    },
    /** 取该档案与场景对应专家的会话：有则复用（继续），无则新建。 */
    async ensureSession(profileId: string, scenarioId: string, context?: ConsultationContext) {
      const expert = scenarioById(scenarioId)?.expert ?? 'natal'
      if (!context) {
        const existing = this.sessions.find(s => s.profile_id === profileId && s.expert === expert)
        if (existing) return existing
      }
      return this.createSession(profileId, scenarioId, context)
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
      try {
        // Detail response owns consultation context; list responses are projections.
        this.activeSession = await (await chatApiLazy()).fetchChatSession(sessionId)
        const page = await (await chatApiLazy()).fetchChatMessages(sessionId)
        this.messages = page.messages
        this.messagesCursor = page.next_cursor
      } catch (error) {
        this.messages = []
        this.messagesCursor = ''
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
      this.messages.push({
        id: assistantId,
        session_id: session.id,
        user_id: session.user_id,
        role: 'assistant',
        kind: 'text',
        content: '',
        created_at: new Date().toISOString(),
      })
      // 从响应式数组取回代理再写入：流式增量与完成落笔必须经代理才会触发
      // ChatThread 重渲染（直改 push 前的原始对象不触发，直连路径曾因此气泡永久为空）。
      const assistant = this.messages[this.messages.length - 1]

      // 流式渲染节流：累积 delta，≤120ms 刷新一次（借命名 chat 实践），
      // 避免高频 delta 触发响应式全量重渲染。
      let pendingText = ''
      let textTimer: ReturnType<typeof setTimeout> | null = null
      const flushText = () => {
        assistant.content = pendingText
        textTimer = null
      }

      try {
        const result = await (await chatApiLazy()).streamChatMessage(session, message, event => {
          if (event.type === 'phase') {
            // phase 只带原始工具名（事实），标签由前端 i18n 渲染。
            this.phase = event.content ? t(toolPhaseKey(event.content)) : this.phase
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
        this.messages = this.messages.filter(item => item.id !== userId && item.id !== assistantId)
        const finalAssistant = result.assistant_message
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
        if (aborted) {
          // 停止生成：保留本地已流出内容并标记「已停止」；部分答案不得静默消失。
          assistant.stopped = true
          return
        }
        reportClientError({
          code: (error as { code?: string } | null)?.code ?? 'chat_send_failed',
          detail: error instanceof Error ? error.message : String(error),
          transport: 'bff',
          agent: session.expert,
          view: 'chat-session',
        })
        this.error = errorMessage(error)
        try {
          const page = await (await chatApiLazy()).fetchChatMessages(session.id)
          this.messages = page.messages
          this.messagesCursor = page.next_cursor
        } catch {
          // 保留本地消息，标记为失败状态
          assistant.error = this.error || t('chat.sendFailed')
        }
      } finally {
        streamController = null
        this.sending = false
      }
    },
    /** 加载更早的一页消息（顶部插入）；返回是否有新内容。 */
    async loadEarlierMessages(): Promise<boolean> {
      const session = this.activeSession
      if (!session || !this.messagesCursor || this.loadingEarlier) return false
      this.loadingEarlier = true
      try {
        const page = await (await chatApiLazy()).fetchChatMessages(session.id, this.messagesCursor)
        this.messages = [...page.messages, ...this.messages]
        this.messagesCursor = page.next_cursor
        return page.messages.length > 0
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.loadingEarlier = false
      }
    },
    stopGeneration() {
      streamController?.abort()
      streamController = null
    },
    resetActive() {
      this.activeSession = null
      this.messages = []
      this.messagesCursor = ''
      this.loadingEarlier = false
      this.error = ''
    },
    reset() {
      this.stopGeneration()
      this.sessions = []
      this.resetActive()
      this.loadingSessions = false
      this.loadingMessages = false
      this.sending = false
      this.loadSessionsError = null
      this.pendingIntent = ''
    },
  },
})
