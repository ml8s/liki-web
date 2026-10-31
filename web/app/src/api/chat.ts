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
  intent: string
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
  intentId: string,
  title?: string,
  context?: ConsultationContext,
  intent?: string,
): Promise<ChatSession> {
  return post<ChatSession>('/api/chat/sessions', {
    profile_id: profileId,
    skill,
    intent: intentId,
    title,
    context,
    ...(intent ? { intent } : {}),
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

export type ChatSessionRef = Pick<ChatSession, 'id' | 'intent'>


export async function streamChatMessage(
  session: ChatSessionRef,
  content: string,
  onEvent: (event: ChatStreamEvent) => void,
  signal?: AbortSignal,
): Promise<SendMessageResult> {
  const messageId = `web-${crypto.randomUUID()}`
  // Product traffic always traverses the BFF: it owns auth, entitlement,
  // session lookup, authoritative history, profile context, and persistence.
  const url = `/api/chat/sessions/${session.id}/ag-ui`
  const forwardedProps = { locale: i18n.global.locale.value }
  const agent = new HttpAgent({
    url,
    threadId: session.id,
    initialMessages: [{ id: messageId, role: 'user' as const, content }],
    fetch: agentFetch,
  })
  const abort = () => agent.abortRun()
  signal?.addEventListener('abort', abort, { once: true })

  let failure: SSEError | null = null
  let completed: unknown = null
  const subscriber: AgentSubscriber = {
    onCustomEvent({ event }) {
      // The BFF emits a non-domain phase fact with the raw tool name. Unknown
      // custom events are ignored for forward compatibility.
      if (event.name === 'hk.liki.phase') {
        if (typeof event.value === 'string' && event.value) {
          onEvent({ type: 'phase', content: event.value })
        }
      }
    },
    onToolCallStartEvent({ event }) {
      if (typeof event.toolCallName === 'string' && event.toolCallName) {
        onEvent({ type: 'phase', content: event.toolCallName })
      }
    },
    onTextMessageContentEvent({ event }) {
      onEvent({ type: 'text-delta', content: event.delta })
    },
    onRunErrorEvent({ event }) {
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
    const payload = (completed ?? run.result) as SendMessageResult | undefined
    if (!payload?.session || !payload.user_message || !payload.assistant_message) {
      throw makeStreamError('Generation stream ended without a final result', 'missing_result')
    }
    return payload
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
