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

  it('retries a transient network failure for idempotent reads', async () => {
    const fetchMock = vi.fn()
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { ok: true } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(request('/api/test')).resolves.toEqual({ ok: true })
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('retries a bad gateway then succeeds', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ title: 'bad gateway' }), { status: 502 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { ok: true } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(request('/api/test')).resolves.toEqual({ ok: true })
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('does not retry a failed non-idempotent request by default', async () => {
    const fetchMock = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
    vi.stubGlobal('fetch', fetchMock)

    await expect(request('/api/test', { method: 'POST', body: '{}' })).rejects.toMatchObject({
      status: 0,
      code: 'network_error',
    })
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('converts timeout to a normalized network failure', async () => {
    vi.stubGlobal('fetch', (_path: string, init?: RequestInit) => new Promise((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => {
        reject(new DOMException('Aborted', 'AbortError'))
      })
    }))

    await expect(request('/api/test', { timeoutMs: 1, retries: 0 })).rejects.toMatchObject({
      status: 0,
      code: 'network_error',
      message: 'Request timed out',
    })
  })

  it('preserves an external abort instead of treating it as a timeout', async () => {
    const controller = new AbortController()
    vi.stubGlobal('fetch', (_path: string, init?: RequestInit) => new Promise((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => {
        reject(new DOMException('Aborted', 'AbortError'))
      })
    }))
    queueMicrotask(() => controller.abort(new DOMException('Caller cancelled', 'AbortError')))

    await expect(request('/api/test', {
      signal: controller.signal,
      timeoutMs: 1_000,
      retries: 0,
    })).rejects.toMatchObject({ name: 'AbortError' })
  })
})
