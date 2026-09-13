import { describe, expect, it, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('loads authenticated user', async () => {
    const user = { id: 'usr_1', email: 'user@example.com', status: 'active', locale: 'zh-Hant' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: user }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    )))
    const store = useAuthStore()
    const loaded = await store.loadMe(true)
    expect(loaded).toEqual(user)
    expect(store.status).toBe('authenticated')
  })

  it('marks unauthenticated on 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: { code: 'unauthorized', message: 'unauthorized' } }),
      { status: 401, headers: { 'Content-Type': 'application/json' } },
    )))
    const store = useAuthStore()
    const loaded = await store.loadMe(true)
    expect(loaded).toBeNull()
    expect(store.status).toBe('unauthenticated')
  })
})
