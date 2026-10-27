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
    runAgent.mockResolvedValue({ result, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))

    await Promise.resolve()
    expect(HttpAgent.lastConfig).toMatchObject({ threadId: 'session_1', url: '/api/chat/sessions/session_1/ag-ui' })
    expect(HttpAgent.lastConfig.initialMessages).toHaveLength(1)
    expect(HttpAgent.lastConfig.initialMessages[0]).toMatchObject({ role: 'user', content: '你好' })

    await mocks.subscriber.onCustomEvent?.({ event: { name: 'agent.runtime.phase', value: 'bazi_chart' } })
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
    expect(returned).toEqual(result)
  })

  it('ignores unknown custom events and empty phase values', async () => {
    runAgent.mockResolvedValue({ result, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))

    await Promise.resolve()
    await mocks.subscriber.onCustomEvent?.({ event: { name: 'hk.liki.unknown', value: 'ignored' } })
    await mocks.subscriber.onCustomEvent?.({ event: { name: 'agent.runtime.phase', value: '' } })
    const returned = await pending
    await mocks.subscriber.onRunFinishedEvent?.({
      outcome: 'success',
      result,
      event: { type: 'RUN_FINISHED' },
    } as any)

    expect(events).toEqual([])
    expect(returned).toEqual(result)
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

  it('maps standard tool-call events to phase labels', async () => {
    runAgent.mockResolvedValue({ result: { ...result, answer: '答案' }, newMessages: [] })
    const events: ChatStreamEvent[] = []
    const pending = streamChatMessage(sessionRef, '你好', event => events.push(event))
    await Promise.resolve()
    await mocks.subscriber.onToolCallStartEvent?.({ event: { toolCallName: 'bazi_chart' } })
    await mocks.subscriber.onTextMessageContentEvent?.({ event: { delta: '答案' } })
    await mocks.subscriber.onRunFinishedEvent?.({ outcome: 'success', result, event: { type: 'RUN_FINISHED' } } as any)
    await pending

    expect(events).toEqual([
      { type: 'phase', content: 'bazi_chart' },
      { type: 'text-delta', content: '答案' },
    ])
  })

})
