import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import type { BirthProfile } from '@/api/birth-profiles'

const profile = {
  id: 'p_1',
  user_id: 'usr_1',
  version: 1,
  nickname: '妈妈',
  name: '',
  birth: {
    gender: 'female',
    date: '1958-05-12',
    time: '06:00',
    instant: '1958-05-11T22:00:00Z',
    location: {
      place: '上海',
      longitude: 121.4737,
      timezone: 'Asia/Shanghai',
    },
  },
  note: '',
  third_party_consent: true,
  last_used_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} satisfies BirthProfile

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
    const store = useBirthProfilesStore()
    await store.loadList(undefined, true)
    expect(store.list).toEqual([profile])
    expect(store.error).toBe('')
  })

  it('creates a profile and prepends it to the list', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: profile }),
      { status: 201, headers: { 'Content-Type': 'application/json' } },
    ))
    const store = useBirthProfilesStore()
    const created = await store.create({
      nickname: '妈妈',
      birth: {
        gender: 'female',
        date: '1958-05-12',
        time: '06:00',
        location: { longitude: 121.4737, timezone: 'Asia/Shanghai' },
      },
    })
    expect(created.id).toBe('p_1')
    expect(store.list[0].id).toBe('p_1')
  })

  it('removes a profile from the list', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { status: 'deleted' } }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    const store = useBirthProfilesStore()
    store.list = [profile]
    await store.remove('p_1')
    expect(store.list).toEqual([])
  })
})
