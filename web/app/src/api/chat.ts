import { HttpAgent, type AgentSubscriber } from '@ag-ui/client'
import { del, get, patch, post, redirectToLogin } from './http'
import { i18n } from '../i18n'
import type { ConsultationContext } from '@shared/domain/consultation'

export interface SSEError extends Error {
  status?: number
  code?: string
}

export type ChatStreamEventType = 'phase' | 'text-delta'

export interface ChatStreamEvent {
  type: ChatStreamEventType
  content?: string
  data?: unknown
}

export interface ChatSession {
  id: string
  user_id: string
  profile_id: string
  skill: string
  expert: string
  title: string
  status: 'active' | 'closed'
  created_at: string
  updated_at: string
  last_message_at: string | null
  context?: ConsultationContext
}

export type ChatMessageKind = 'text' | 'system_profile'

export interface ChatMessage {
  id: string
  session_id: string
  user_id: string
  role: 'user' | 'assistant'
  kind: ChatMessageKind
  content: string
  created_at: string
  error?: string
  /** 仅前端本地标记：用户主动停止生成，保留已流出的部分内容 */
  stopped?: boolean
}

export function fetchChatSessions(): Promise<ChatSession[]> {
  return get<ChatSession[]>('/api/chat/sessions')
}

export function fetchChatSession(sessionId: string): Promise<ChatSession> {
  return get<ChatSession>(`/api/chat/sessions/${sessionId}`)
}

export function createChatSession(
  profileId: string,
  skill: string,
  expert: string,
  title?: string,
  context?: ConsultationContext,
): Promise<ChatSession> {
  return post<ChatSession>('/api/chat/sessions', {
    profile_id: profileId,
    skill,
    expert,
    title,
    context,
  })
}

export function renameChatSession(sessionId: string, title: string): Promise<ChatSession> {
  return patch<ChatSession>(`/api/chat/sessions/${sessionId}`, { title })
}

export function deleteChatSession(sessionId: string): Promise<{ status: string }> {
  return del<{ status: string }>(`/api/chat/sessions/${sessionId}`)
}

export interface ChatMessagesPage {
  messages: ChatMessage[]
  /** 非空表示还有更早的消息，可作为下一次请求的 before 游标。 */
  next_cursor: string
}

export function fetchChatMessages(sessionId: string, before?: string): Promise<ChatMessagesPage> {
  const query = before ? `?before=${encodeURIComponent(before)}` : ''
  return get<ChatMessagesPage>(`/api/chat/sessions/${sessionId}/messages${query}`)
}

export interface SendMessageResult {
  session: ChatSession
  active_skill: string
  user_message: ChatMessage
  assistant_message: ChatMessage
}

/**
 * 完成态归一化：中继返回 liki-web 的权威消息对象；直连只返回 runtime 的
 * 权威回答文本（历史由运行完成回调落库）。
 */
export type ChatCompletion =
  | { transport: 'relay'; result: SendMessageResult }
  | { transport: 'direct'; answer: string }

export interface ChatSessionRef {
  id: string
  expert: string
}

/** 一轮既有对话（直连时必须由客户端携带：runtime 无状态）。 */
export interface ChatTurn {
  id: string
  role: 'user' | 'assistant'
  content: string
}

/**
 * 出生档案上下文（契约 C3 forwardedProps.profile；字段与 liki-web 的
 * AgentProfile 一一对应，刻意不含身份/联系方式/备注/数据库 id）。
 */
export interface AgentProfile {
  gender: string
  instant?: string | null
  location?: {
    longitude?: number | null
    timezone?: string | null
  } | null
}

export interface DirectChatContext {
  history?: ChatTurn[]
  profile?: AgentProfile | null
  consultationContext?: ConsultationContext
}

export async function streamChatMessage(
  session: ChatSessionRef,
  content: string,
  onEvent: (event: ChatStreamEvent) => void,
  signal?: AbortSignal,
  context?: DirectChatContext,
): Promise<ChatCompletion> {
  const messageId = `web-${crypto.randomUUID()}`
  // 传输面开关：构建期注入 VITE_CHAT_TRANSPORT=direct 时走直连（Caddy mTLS），
  // 否则保持中继（回滚杆：部署上一版镜像即回到中继）。
  const directTransport = import.meta.env.VITE_CHAT_TRANSPORT === 'direct'
  const url = directTransport
    ? `/rt/${session.expert}/ag-ui`
    : `/api/chat/sessions/${session.id}/ag-ui`
  // 直连的 runtime 是无状态的：中继时代由 BFF 从数据库补上的对话历史与出生
  // 档案上下文，必须由客户端随请求携带（契约 C3）。中继分支忽略 context。
  // 按角色窄化：AG-UI 的 Message 是可辨识联合，role 为宽联合的对象字面量
  // 无法直接分配到联合成员上。
  const historyMessages = (context?.history ?? []).map(turn => (
    turn.role === 'user'
      ? { id: turn.id, role: 'user' as const, content: turn.content }
      : { id: turn.id, role: 'assistant' as const, content: turn.content }
  ))
  const initialMessages = [...historyMessages, { id: messageId, role: 'user' as const, content }]
  // SDK 只在 runAgent 参数读取 forwardedProps（构造器配置被忽略）。中继的
  // gateway 从 locale 取界面语言，直连额外携带 product/profile（契约 C3）。
  const forwardedProps: Record<string, unknown> = { product: 'liki', locale: i18n.global.locale.value }
  if (context?.profile) {
    forwardedProps.profile = context.profile
  }
  if (context?.consultationContext) {
    forwardedProps.consultation_context = context.consultationContext
  }
  const agent = new HttpAgent({
    url,
    threadId: session.id,
    initialMessages,
    fetch: agentFetch,
  })
  const abort = () => agent.abortRun()
  signal?.addEventListener('abort', abort, { once: true })

  let failure: SSEError | null = null
  let completed: unknown = null
  const subscriber: AgentSubscriber = {
    onCustomEvent({ event }) {
      // 反域名自定义事件只承载事实：phase 带原始工具名。
      // 未知自定义事件一律忽略（向前兼容）。
      if (event.name === 'hk.liki.phase') {
        if (typeof event.value === 'string' && event.value) {
          onEvent({ type: 'phase', content: event.value })
        }
      }
    },
    onToolCallStartEvent({ event }) {
      // 直连模式：runtime 直接发标准 AG-UI 工具事件（没有 hk.liki.phase），
      // 工具名同样交给前端 i18n 渲染阶段标签。
      if (typeof event.toolCallName === 'string' && event.toolCallName) {
        onEvent({ type: 'phase', content: event.toolCallName })
      }
    },
    onTextMessageContentEvent({ event }) {
      onEvent({ type: 'text-delta', content: event.delta })
    },
    onRunErrorEvent({ event }) {
      // 协议错误码优先（前端按 errors.code.* 本地化渲染）；中继路径无码时回落。
      failure = makeStreamError(event.message || 'Generation failed', event.code || 'generation_failed')
    },
    onRunFinishedEvent(params) {
      if (params.outcome !== 'success') {
        failure = makeStreamError('Generation canceled', 'generation_cancelled')
        return
      }
      completed = params.result
    },
  }

  try {
    const run = await agent.runAgent({
      runId: crypto.randomUUID(),
      forwardedProps,
    }, subscriber)
    if (failure) throw failure
    if (directTransport) {
      const answer = typeof (completed as { answer?: unknown } | null)?.answer === 'string'
        ? String((completed as { answer: unknown }).answer)
        : ''
      if (!answer) {
        throw makeStreamError('Generation stream ended without a final result', 'missing_result')
      }
      return { transport: 'direct', answer }
    }
    const payload = (completed ?? run.result) as SendMessageResult | undefined
    if (!payload?.session || !payload.user_message || !payload.assistant_message) {
      throw makeStreamError('Generation stream ended without a final result', 'missing_result')
    }
    return { transport: 'relay', result: payload }
  } catch (cause) {
    if (cause instanceof Error && cause.name === 'AbortError') throw cause
    if (failure) throw failure
    if (cause instanceof TypeError) {
      throw makeStreamError('Network request failed', 'network_error', 0)
    }
    throw cause instanceof Error ? cause : makeStreamError('Generation failed', 'generation_failed')
  } finally {
    signal?.removeEventListener('abort', abort)
  }
}

async function agentFetch(url: string, init: RequestInit): Promise<Response> {
  let response: Response
  try {
    response = await fetch(url, { ...init, credentials: 'same-origin' })
  } catch {
    throw makeStreamError('Network request failed', 'network_error', 0)
  }
  if (response.ok) return response
  if (response.status === 401) redirectToLogin()
  // 错误体统一按 RFC 9457 problem+json 解析：detail 是消息，code 是扩展成员
  // （runtime）；liki-web 的 type 也承载 code（兼容保留）。legacy 的
  // { error: { message, code } } 仅作兜底。
  const envelope = await response.json().catch(() => null) as {
    detail?: string
    title?: string
    code?: string
    type?: string
    error?: { message?: string; code?: string }
  } | null
  throw makeStreamError(
    envelope?.detail ?? envelope?.error?.message ?? envelope?.title ?? `HTTP ${response.status}`,
    envelope?.code ?? envelope?.error?.code ?? envelope?.type ?? 'request_failed',
    response.status,
  )
}

function makeStreamError(message: string, code?: string, status?: number): SSEError {
  const error = new Error(message) as SSEError
  error.code = code
  error.status = status
  return error
}
