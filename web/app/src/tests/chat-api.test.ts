import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  const runAgent = vi.fn()
  class HttpAgent {
    static lastConfig: any
    abortRun = vi.fn()

    constructor(config: any) {
      HttpAgent.lastConfig = config
    }

    async runAgent(_parameters: any, nextSubscriber: any) {
      mocks.subscriber = nextSubscriber
      return runAgent(_parameters, nextSubscriber)
    }
  }
  return { runAgent, HttpAgent, subscriber: undefined as any }
})

const runAgent = mocks.runAgent
const HttpAgent = mocks.HttpAgent

vi.mock('@ag-ui/client', () => ({ HttpAgent: mocks.HttpAgent }))

import { streamChatMessage, type ChatStreamEvent } from '@/api/chat'

const sessionRef = { id: 'session_1', expert: 'natal' }

const result = {
  session: { id: 'session_1' },
  active_skill: 'liki',
  user_message: { id: 'message_1' },
  assistant_message: { id: 'message_2', content: 'answer' },
} as any

beforeEach(() => {
  runAgent.mockReset()
  mocks.subscriber = undefined
})

describe('streamChatMessage with AG-UI HttpAgent', () => {
  it('maps AG-UI events and returns RUN_FINISHED result', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'relay')
    runAgent.mockResolvedValue({ result, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))

    await Promise.resolve()
    expect(HttpAgent.lastConfig).toMatchObject({ threadId: 'session_1', url: '/api/chat/sessions/session_1/ag-ui' })
    expect(HttpAgent.lastConfig.initialMessages).toHaveLength(1)
    expect(HttpAgent.lastConfig.initialMessages[0]).toMatchObject({ role: 'user', content: '你好' })

    await mocks.subscriber.onCustomEvent?.({ event: { name: 'hk.liki.phase', value: 'bazi_chart' } })
    await mocks.subscriber.onTextMessageContentEvent?.({ event: { delta: '答案' } })
    const returned = await pending
    await mocks.subscriber.onRunFinishedEvent?.({
      outcome: 'success',
      result,
      event: { type: 'RUN_FINISHED' },
    } as any)

    expect(events).toEqual([
      { type: 'phase', content: 'bazi_chart' },
      { type: 'text-delta', content: '答案' },
    ])
    expect(returned).toEqual({ transport: 'relay', result })
  })

  it('ignores unknown custom events and empty phase values', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'relay')
    runAgent.mockResolvedValue({ result, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))

    await Promise.resolve()
    await mocks.subscriber.onCustomEvent?.({ event: { name: 'hk.liki.unknown', value: 'ignored' } })
    await mocks.subscriber.onCustomEvent?.({ event: { name: 'hk.liki.phase', value: '' } })
    const returned = await pending
    await mocks.subscriber.onRunFinishedEvent?.({
      outcome: 'success',
      result,
      event: { type: 'RUN_FINISHED' },
    } as any)

    expect(events).toEqual([])
    expect(returned).toEqual({ transport: 'relay', result })
  })

  it('throws RUN_ERROR without inventing a completed result', async () => {
    runAgent.mockResolvedValue({ result: null, newMessages: [] })
    const pending = streamChatMessage(sessionRef, '你好', () => {})
    await Promise.resolve()
    await mocks.subscriber.onRunErrorEvent?.({ event: { message: 'engine unavailable' } })

    await expect(pending).rejects.toThrow('engine unavailable')
  })

  it('carries the protocol error code for localized rendering', async () => {
    runAgent.mockResolvedValue({ result: null, newMessages: [] })
    const pending = streamChatMessage(sessionRef, '你好', () => {})
    await Promise.resolve()
    await mocks.subscriber.onRunErrorEvent?.({
      event: { message: 'agent run exceeded its deadline', code: 'runtime_timeout' },
    })

    await expect(pending).rejects.toMatchObject({ code: 'runtime_timeout' })
  })

  it('maps standard tool-call events to phase labels on the direct transport', async () => {
    // direct 分支契约：RUN_FINISHED.result 必须携带 answer。
    runAgent.mockResolvedValue({ result: { ...result, answer: '答案' }, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))
    await Promise.resolve()
    // 直连模式下 runtime 直接发标准 AG-UI 工具事件（没有 hk.liki.phase）。
    await mocks.subscriber.onToolCallStartEvent?.({ event: { toolCallName: 'bazi_chart' } })
    await mocks.subscriber.onTextMessageContentEvent?.({ event: { delta: '答案' } })
    await mocks.subscriber.onRunFinishedEvent?.({ outcome: 'success', result, event: { type: 'RUN_FINISHED' } } as any)
    await pending

    expect(events).toEqual([
      { type: 'phase', content: 'bazi_chart' },
      { type: 'text-delta', content: '答案' },
    ])
  })

  it('uses the direct-connect route and returns the runtime answer shape', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'direct')
    runAgent.mockResolvedValue({ result: { answer: '答案' }, newMessages: [] })
    const pending = streamChatMessage(sessionRef, '你好', () => {})
    await Promise.resolve()
    expect(HttpAgent.lastConfig).toMatchObject({ url: '/rt/natal/ag-ui', threadId: 'session_1' })
    await mocks.subscriber.onRunFinishedEvent?.({ outcome: 'success', result: { answer: '答案' }, event: { type: 'RUN_FINISHED' } } as any)
    const returned = await pending
    expect(returned).toEqual({ transport: 'direct', answer: '答案' })
    vi.unstubAllEnvs()
  })
})
