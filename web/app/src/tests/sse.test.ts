import { describe, expect, it, vi } from 'vitest'
import { streamRequest } from '@/api/sse'

function sseResponse(frames: string[]) {
  const encoder = new TextEncoder()
  let index = 0
  return new Response(new ReadableStream({
    pull(controller) {
      if (index < frames.length) {
        controller.enqueue(encoder.encode(frames[index++]))
      } else {
        controller.close()
      }
    },
  }), { status: 200, headers: { 'Content-Type': 'text/event-stream' } })
}

describe('streamRequest', () => {
  it('parses SSE data frames', async () => {
    const events: unknown[] = []
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(sseResponse([
      'data: {"type":"thinking"}\n\n',
      'data: {"type":"text-delta","content":"你好"}\n\n',
    ])))

    await expect(streamRequest('/api/test', {}, event => events.push(event))).resolves.toBeUndefined()
    expect(events).toEqual([
      { type: 'thinking' },
      { type: 'text-delta', content: '你好' },
    ])
  })
})
