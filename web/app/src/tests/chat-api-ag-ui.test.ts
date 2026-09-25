import { describe, expect, it, vi } from 'vitest'
import { streamChatMessage } from '@/api/chat'

const sseEvent = (payload: Record<string, unknown>) => `data: ${JSON.stringify(payload)}\n\n`

describe('streamChatMessage against the official HttpAgent parser', () => {
  it('parses an AG-UI SSE stream and returns RUN_FINISHED.result', async () => {
    const body = [
      sseEvent({ type: 'RUN_STARTED', threadId: 'session_1', runId: 'run_1' }),
      sseEvent({ type: 'CUSTOM', name: 'liki.phase', value: '排盘中' }),
      sseEvent({ type: 'TEXT_MESSAGE_START', messageId: 'assistant', role: 'assistant' }),
      sseEvent({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'assistant', delta: '答案' }),
      sseEvent({ type: 'TEXT_MESSAGE_END', messageId: 'assistant' }),
      sseEvent({
        type: 'RUN_FINISHED',
        threadId: 'session_1',
        runId: 'run_1',
        outcome: { type: 'success' },
        result: {
          session: { id: 'session_1' },
          active_skill: 'liki',
          user_message: { id: 'user_1' },
          assistant_message: { id: 'assistant_1', content: '答案' },
        },
      }),
      '',
    ].join('\n')
    const fetchSpy = vi.fn().mockResolvedValue(new Response(body, {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }))
    vi.stubGlobal('fetch', fetchSpy)

    const events: unknown[] = []
    const result = await streamChatMessage('session_1', '你好', event => events.push(event))
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/chat/sessions/session_1/ag-ui',
      expect.objectContaining({ method: 'POST', credentials: 'same-origin' }),
    )
    expect(events).toEqual([
      { type: 'phase', content: '排盘中' },
      { type: 'text-delta', content: '答案' },
    ])
    expect(result.assistant_message).toEqual({ id: 'assistant_1', content: '答案' })
    vi.unstubAllGlobals()
  })
})
