import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAdminStore } from '@/stores/admin'

describe('admin store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('loads effective capabilities and scopes navigation helpers', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { email: 'ops@example.com', capabilities: ['entitlement.read'] } }),
      { status: 200 },
    )))
    const store = useAdminStore()
    await expect(store.loadSession(true)).resolves.toBe(true)
    expect(store.status).toBe('ready')
    expect(store.can('entitlement.read')).toBe(true)
    expect(store.can('payout.read')).toBe(false)
    expect(store.canAny(['payout.read', 'entitlement.read'])).toBe(true)
  })

  it('marks authenticated accounts without capabilities as denied', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { email: 'user@example.com', capabilities: [] } }),
      { status: 200 },
    )))
    const store = useAdminStore()
    await expect(store.loadSession(true)).resolves.toBe(false)
    expect(store.status).toBe('denied')
  })
})
