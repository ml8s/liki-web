import { HttpAgent, type AgentSubscriber } from '@ag-ui/client'
import { del, get, patch, post, redirectToLogin } from './http'
import { i18n } from '../i18n'

export interface SSEError extends Error {
  status?: number
  code?: string
}

export type ChatStreamEventType = 'thinking' | 'phase' | 'thinking-delta' | 'text-delta' | 'error' | 'done'

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
  /** 仅本地流式期间附加的推理内容（thinking-delta），服务端不回存 */
  thinking?: string
  error?: string
}

export function fetchChatSessions(): Promise<ChatSession[]> {
  return get<ChatSession[]>('/api/chat/sessions')
}

export function fetchChatSession(sessionId: string): Promise<ChatSession> {
  return get<ChatSession>(`/api/chat/sessions/${sessionId}`)
}

export function createChatSession(profileId: string, skill: string, expert: string, title?: string): Promise<ChatSession> {
  return post<ChatSession>('/api/chat/sessions', { profile_id: profileId, skill, expert, title })
}

export function renameChatSession(sessionId: string, title: string): Promise<ChatSession> {
  return patch<ChatSession>(`/api/chat/sessions/${sessionId}`, { title })
}

export function deleteChatSession(sessionId: string): Promise<{ status: string }> {
  return del<{ status: string }>(`/api/chat/sessions/${sessionId}`)
}

export function fetchChatMessages(sessionId: string): Promise<ChatMessage[]> {
  return get<ChatMessage[]>(`/api/chat/sessions/${sessionId}/messages`)
}

export interface SendMessageResult {
  session: ChatSession
  active_skill: string
  user_message: ChatMessage
  assistant_message: ChatMessage
}

export async function streamChatMessage(
  sessionId: string,
  content: string,
  onEvent: (event: ChatStreamEvent) => void,
  signal?: AbortSignal,
): Promise<SendMessageResult> {
  const messageId = `web-${crypto.randomUUID()}`
  const agent = new HttpAgent({
    url: `/api/chat/sessions/${sessionId}/ag-ui`,
    threadId: sessionId,
    initialMessages: [{ id: messageId, role: 'user', content }],
    fetch: agentFetch,
  })
  const abort = () => agent.abortRun()
  signal?.addEventListener('abort', abort, { once: true })

  let failure: SSEError | null = null
  let completed: SendMessageResult | null = null
  const subscriber: AgentSubscriber = {
    onCustomEvent({ event }) {
      const type = event.name === 'liki.thinking' ? 'thinking-delta' : 'phase'
      onEvent({ type, content: typeof event.value === 'string' ? event.value : undefined })
    },
    onTextMessageContentEvent({ event }) {
      onEvent({ type: 'text-delta', content: event.delta })
    },
    onRunErrorEvent({ event }) {
      failure = makeStreamError(event.message || 'Generation failed', 'generation_failed')
    },
    onRunFinishedEvent(params) {
      if (params.outcome !== 'success') {
        failure = makeStreamError('Generation canceled', 'generation_cancelled')
        return
      }
      completed = params.result as SendMessageResult
    },
  }

  try {
    const run = await agent.runAgent({
      runId: crypto.randomUUID(),
      forwardedProps: { locale: i18n.global.locale.value },
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
  const envelope = await response.json().catch(() => null) as { error?: { message?: string; code?: string } } | null
  throw makeStreamError(
    envelope?.error?.message ?? `HTTP ${response.status}`,
    envelope?.error?.code ?? 'request_failed',
    response.status,
  )
}

function makeStreamError(message: string, code?: string, status?: number): SSEError {
  const error = new Error(message) as SSEError
  error.code = code
  error.status = status
  return error
}
