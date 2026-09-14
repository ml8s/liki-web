import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useEntitlementsStore } from '@/stores/entitlements'

beforeEach(() => {
  setActivePinia(createPinia())
  vi.restoreAllMocks()
  global.fetch = vi.fn().mockResolvedValue(new Response(
    JSON.stringify({ data: { entitlements: [], allowed_skills: ['liki-naming', 'liki-bazi'] } }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  ))
})

describe('entitlements store', () => {
  it('loads backend-authorized skills once', async () => {
    const store = useEntitlementsStore()
    await store.load()
    await store.load()

    expect(store.allowedSkills).toEqual(['liki-naming', 'liki-bazi'])
    expect(store.hasSkill('liki-bazi')).toBe(true)
    expect(store.hasSkill('liki-fengshui')).toBe(false)
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })
})
