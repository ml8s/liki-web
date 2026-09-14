import { describe, expect, it, vi } from 'vitest'
import { fetchSessions, revokeSession, revokeAllSessions, type SessionInfo } from '@/api/me'

const sessions = [
  {
    id: 'sess_1',
    created_at: '2026-01-01T00:00:00Z',
    expires_at: '2026-02-01T00:00:00Z',
    last_seen_at: '2026-01-01T12:00:00Z',
    ip_hash: 'hash-a',
    user_agent: 'Mozilla/5.0 (iPhone)',
    is_current: true,
  },
] satisfies SessionInfo[]

describe('me sessions API', () => {
  it('fetches session list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: sessions }),
      { status: 200 },
    )))
    await expect(fetchSessions()).resolves.toEqual(sessions)
  })

  it('revokes a single session with DELETE', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { status: 'revoked' } }),
      { status: 200 },
    ))
    vi.stubGlobal('fetch', fetchMock)
    await expect(revokeSession('sess_1')).resolves.toEqual({ status: 'revoked' })
    expect(fetchMock).toHaveBeenCalledWith('/api/me/sessions/sess_1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('revokes all other sessions', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { status: 'revoked', count: 2 } }),
      { status: 200 },
    )))
    await expect(revokeAllSessions()).resolves.toEqual({ status: 'revoked', count: 2 })
  })
})