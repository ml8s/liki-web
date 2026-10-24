import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useChatStore } from '@/stores/chat'
import { useSalesStore } from '@/stores/sales-store'
import type { BirthProfile } from '@/api/birth-profiles'

function profile(id: string): BirthProfile {
  return {
    id,
    user_id: 'user_1',
    version: 1,
    nickname: id,
    name: '',
    birth: {
      gender: 'female',
      date: '1990-01-01',
      time: '10:00',
      instant: '1990-01-01T02:00:00Z',
      location: { place: '上海', longitude: 121.4726, timezone: 'Asia/Shanghai' },
    },
    note: '',
    third_party_consent: true,
    last_used_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }
}

describe('sales enterInterpretation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('creates the interpretation session for the selected profile', async () => {
    const profiles = useBirthProfilesStore()
    profiles.list = [profile('profile_first'), profile('profile_selected')]
    const createSession = vi.fn().mockResolvedValue({ id: 'session_selected' })
    useChatStore().createSession = createSession

    const sessionId = await useSalesStore().enterInterpretation('profile_selected')

    expect(sessionId).toBe('session_selected')
    expect(createSession).toHaveBeenCalledWith('profile_selected', 'natal', {
      version: 1,
      scenario: 'natal',
      birth: {
        gender: 'female',
        date: '1990-01-01',
        time: '10:00',
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
    })
  })

  it('falls back to the first profile when none is selected', async () => {
    const profiles = useBirthProfilesStore()
    profiles.list = [profile('profile_first'), profile('profile_second')]
    const createSession = vi.fn().mockResolvedValue({ id: 'session_first' })
    useChatStore().createSession = createSession

    const sessionId = await useSalesStore().enterInterpretation()

    expect(sessionId).toBe('session_first')
    expect(createSession).toHaveBeenCalledWith('profile_first', 'natal', {
      version: 1,
      scenario: 'natal',
      birth: {
        gender: 'female',
        date: '1990-01-01',
        time: '10:00',
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
    })
  })
})
