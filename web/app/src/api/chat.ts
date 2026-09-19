import { get, post } from './http'
import { streamRequest, type SSEError } from './sse'

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

export function createChatSession(profileId: string, skill: string, title?: string): Promise<ChatSession> {
  return post<ChatSession>('/api/chat/sessions', { profile_id: profileId, skill, title })
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
  let result: SendMessageResult | null = null
  await streamRequest(
    `/api/chat/sessions/${sessionId}/messages`,
    { content },
    (rawEvent: { type: string; content?: string; data?: unknown }) => {
      const event = rawEvent as ChatStreamEvent
      if (event.type === 'error') {
        const error = new Error(event.content || '生成失败') as SSEError
        error.code = 'generation_failed'
        throw error
      }
      if (event.type === 'done') {
        result = event.data as SendMessageResult
      }
      onEvent(event)
    },
    signal,
  )
  if (!result) {
    throw new Error('流式响应缺少完成事件')
  }
  return result
}
