import { describe, expect, it } from 'vitest'
import { bffChatStream } from '../e2e/helpers/chat'

describe('BFF AG-UI E2E stream fixture', () => {
  it('returns the product completion shape rather than the runtime answer shape', () => {
    const session = {
      id: 'session_bff',
      user_id: 'user_1',
      skill: 'liki',
      expert: 'natal',
    }
    const body = bffChatStream({
      session,
      answer: '权威答案',
      question: '用户问题',
      runId: 'run_1',
    })

    const events = body.trim().split('\n\n').map(line => JSON.parse(line.replace(/^data: /, '')))
    const finished = events.at(-1)
    expect(finished).toMatchObject({
      type: 'RUN_FINISHED',
      threadId: 'session_bff',
      runId: 'run_1',
      outcome: { type: 'success' },
    })
    expect(finished.result).toMatchObject({
      session,
      active_skill: 'liki',
      user_message: {
        session_id: 'session_bff',
        role: 'user',
        content: '用户问题',
      },
      assistant_message: {
        session_id: 'session_bff',
        role: 'assistant',
        content: '权威答案',
      },
    })
    expect(finished.result).not.toHaveProperty('answer')
  })
})
