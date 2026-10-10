import { describe, expect, it, vi } from 'vitest'
import { request, ApiError } from '@/api/http'

describe('http client', () => {
  it('unwraps API envelope', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { ok: true } }),
      { status: 200 },
    )))
    await expect(request('/api/test')).resolves.toEqual({ ok: true })
  })

  it('throws normalized API error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ type: 'invalid', title: 'invalid', status: 400, detail: 'bad request' }),
      { status: 400 },
    )))
    await expect(request('/api/test')).rejects.toMatchObject({
      status: 400,
      code: 'invalid',
      message: 'bad request',
    } satisfies Partial<ApiError>)
  })
})
