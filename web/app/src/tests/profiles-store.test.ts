import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useProfilesStore } from '@/stores/profiles'
import type { Profile } from '@/api/profiles'

const profile = {
  id: 'p_1',
  user_id: 'usr_1',
  version: 1,
  nickname: '妈妈',
  name: '',
  gender: 'female',
  calendar: 'lunar',
  birth_date: '1958-闰5-12',
  birth_time: '06:00',
  birth_time_known: 'known',
  birth_place: '上海',
  note: '',
  third_party_consent: true,
  last_used_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} satisfies Profile

beforeEach(() => {
  setActivePinia(createPinia())
  vi.restoreAllMocks()
  global.fetch = vi.fn()
})

describe('profiles store', () => {
  it('loads and stores the profile list', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { profiles: [profile], next_cursor: '' } }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    const store = useProfilesStore()
    await store.loadList(undefined, true)
    expect(store.list).toEqual([profile])
    expect(store.error).toBe('')
  })

  it('creates a profile and prepends it to the list', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: profile }),
      { status: 201, headers: { 'Content-Type': 'application/json' } },
    ))
    const store = useProfilesStore()
    const created = await store.create({
      nickname: '妈妈', gender: 'female', calendar: 'lunar',
      birth_date: '1958-闰5-12', birth_time: '06:00', birth_time_known: 'known',
    })
    expect(created.id).toBe('p_1')
    expect(store.list[0].id).toBe('p_1')
  })

  it('removes a profile from the list', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { status: 'deleted' } }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    const store = useProfilesStore()
    store.list = [profile]
    await store.remove('p_1')
    expect(store.list).toEqual([])
  })
})