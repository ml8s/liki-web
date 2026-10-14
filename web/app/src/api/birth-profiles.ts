import { get, post, request } from './http'
import { i18n } from '@/i18n'

export interface BirthProfile {
  id: string
  user_id: string
  version: number
  nickname: string
  name: string
  gender: 'male' | 'female'
  calendar: 'solar' | 'lunar'
  birth_date: string
  birth_time: string
  birth_time_known: 'known' | 'approximate' | 'unknown'
  birth_place: string
  note: string
  third_party_consent: boolean
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface BirthProfileInput {
  nickname: string
  name?: string
  gender: 'male' | 'female'
  calendar: 'solar' | 'lunar'
  birth_date: string
  birth_time: string
  birth_time_known: 'known' | 'approximate' | 'unknown'
  birth_place?: string
  note?: string
  third_party_consent?: boolean
}

export interface BirthProfileListResult {
  profiles: BirthProfile[]
  next_cursor: string
}

export function fetchBirthProfiles(cursor?: string): Promise<BirthProfileListResult> {
  const query = new URLSearchParams({ limit: '50' })
  if (cursor) query.set('cursor', cursor)
  return get<BirthProfileListResult>(`/api/birth-profiles?${query.toString()}`)
}

export function fetchBirthProfile(id: string): Promise<BirthProfile> {
  return get<BirthProfile>(`/api/birth-profiles/${id}`)
}

export function createBirthProfile(input: BirthProfileInput): Promise<BirthProfile> {
  return post<BirthProfile>('/api/birth-profiles', input)
}

export function updateBirthProfile(id: string, expectedVersion: number, input: BirthProfileInput): Promise<BirthProfile> {
  return request<BirthProfile>(`/api/birth-profiles/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ expected_version: expectedVersion, ...input }),
  })
}

export function deleteBirthProfile(id: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/birth-profiles/${id}`, { method: 'DELETE' })
}

export function birthSummary(profile: BirthProfile): string {
  const t = i18n.global.t
  const gender = profile.gender === 'male' ? t('profiles.male') : t('profiles.female')
  const calendar = profile.calendar === 'lunar' ? t('profiles.form.lunar') : t('profiles.form.solar')
  const known = {
    known: t('chat.systemBirthProfile.known'),
    approximate: t('chat.systemBirthProfile.approximate'),
    unknown: t('chat.systemBirthProfile.unknown'),
  }[profile.birth_time_known]
  const place = profile.birth_place ? ` · ${profile.birth_place}` : ''
  return i18n.global.t('profiles.summary', {
    calendar,
    date: profile.birth_date,
    time: profile.birth_time || '',
    gender,
    known,
    place,
  })
}
