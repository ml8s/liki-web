import { describe, expect, it, vi } from 'vitest'
import { streamChatMessage } from '@/api/chat'

const sseEvent = (payload: Record<string, unknown>) => `data: ${JSON.stringify(payload)}\n\n`

describe('streamChatMessage against the official HttpAgent parser', () => {
  it('parses an AG-UI SSE stream and returns RUN_FINISHED.result', async () => {
    // 中继形态的 RUN_FINISHED（无 answer 字段）：显式钉住被弃用的中继路径。
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'relay')
    const body = [
      sseEvent({ type: 'RUN_STARTED', threadId: 'session_1', runId: 'run_1' }),
      sseEvent({ type: 'CUSTOM', name: 'hk.liki.phase', value: 'bazi_chart' }),
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
    const result = await streamChatMessage({ id: 'session_1', expert: 'natal' }, '你好', event => events.push(event))
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/chat/sessions/session_1/ag-ui',
      expect.objectContaining({ method: 'POST', credentials: 'same-origin' }),
    )
    expect(events).toEqual([
      { type: 'phase', content: 'bazi_chart' },
      { type: 'text-delta', content: '答案' },
    ])
    expect(result).toEqual({
      transport: 'relay',
      result: expect.objectContaining({
        assistant_message: { id: 'assistant_1', content: '答案' },
      }),
    })
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })

  it('normalizes the runtime completion on the direct transport', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'direct')
    const body = [
      sseEvent({ type: 'RUN_STARTED', threadId: 'session_1', runId: 'run_1' }),
      sseEvent({ type: 'TOOL_CALL_START', toolCallId: 'call_1', toolCallName: 'bazi_chart' }),
      sseEvent({ type: 'TOOL_CALL_END', toolCallId: 'call_1' }),
      sseEvent({ type: 'TEXT_MESSAGE_START', messageId: 'assistant', role: 'assistant' }),
      sseEvent({ type: 'TEXT_MESSAGE_CONTENT', messageId: 'assistant', delta: '答案' }),
      sseEvent({ type: 'TEXT_MESSAGE_END', messageId: 'assistant' }),
      sseEvent({
        type: 'RUN_FINISHED',
        threadId: 'session_1',
        runId: 'run_1',
        outcome: { type: 'success' },
        result: { answer: '答案' },
      }),
      '',
    ].join('\n')
    const fetchSpy = vi.fn().mockResolvedValue(new Response(body, {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }))
    vi.stubGlobal('fetch', fetchSpy)

    const events: unknown[] = []
    const result = await streamChatMessage(
      { id: 'session_1', expert: 'divination' },
      '继续',
      event => events.push(event),
      undefined,
      {
        history: [
          { id: 'm1', role: 'user', content: '你好' },
          { id: 'm2', role: 'assistant', content: '你好，请提供信息。' },
        ],
      profile: {
        gender: 'male',
      },
      consultationContext: {
        version: 1,
        scenario: 'divination',
        question: '這個項目能成嗎',
        casting: {
          mode: 'auto',
          order: 'bottom_up',
          yaos: [7, 8, 8, 7, 8, 9],
          dong_yao: [6],
          casting_id: 'cast_1',
        },
        casting_instant: '2026-10-23T12:00:00Z',
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
      },
    )

    expect(events).toEqual([
      { type: 'phase', content: 'bazi_chart' },
      { type: 'text-delta', content: '答案' },
    ])
    expect(result).toEqual({ transport: 'direct', answer: '答案' })

    // 直连必须携带历史与档案上下文（runtime 无状态；中继由 BFF 从数据库补）。
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit]
    expect(String(url)).toContain('/rt/divination/ag-ui')
    const requestBody = JSON.parse(String(init.body)) as {
      messages: Array<{ id: string; role: string; content: string }>
      forwardedProps: Record<string, unknown>
    }
    expect(requestBody.messages).toEqual([
      { id: 'm1', role: 'user', content: '你好' },
      { id: 'm2', role: 'assistant', content: '你好，请提供信息。' },
      expect.objectContaining({ role: 'user', content: '继续' }),
    ])
    expect(requestBody.forwardedProps.product).toBe('liki')
    expect(requestBody.forwardedProps.profile).toEqual({
      gender: 'male',
    })
    expect(requestBody.forwardedProps.consultation_context).toEqual({
      version: 1,
      scenario: 'divination',
      question: '這個項目能成嗎',
      casting: {
        mode: 'auto',
        order: 'bottom_up',
        yaos: [7, 8, 8, 7, 8, 9],
        dong_yao: [6],
        casting_id: 'cast_1',
      },
      casting_instant: '2026-10-23T12:00:00Z',
      location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
    })
    expect(typeof requestBody.forwardedProps.locale).toBe('string')

    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
  })
})
