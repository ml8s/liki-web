import { get, post, request } from './http'
import { i18n } from '@/i18n'
import { formatInstant } from '@shared/domain/time'

export interface BirthProfileLocation {
  place: string
  place_id?: string
  longitude: number | null
  timezone: string | null
}

export interface BirthFacts {
  gender: 'male' | 'female'
  date: string | null
  time: string | null
  location: BirthProfileLocation
  instant: string | null
}

export interface BirthProfile {
  id: string
  user_id: string
  version: number
  nickname: string
  name: string
  birth: BirthFacts
  note: string
  third_party_consent: boolean
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface BirthProfileInput {
  nickname: string
  name?: string
  birth: {
    gender: 'male' | 'female'
    date: string
    time: string
    location: {
      place?: string
      place_id?: string
      longitude: number
      timezone: string
    }
  }
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
  const gender = profile.birth.gender === 'male' ? t('profiles.male') : t('profiles.female')
  const place = profile.birth.location.place ? ` · ${profile.birth.location.place}` : ''
  const canonicalTime = profile.birth.instant && profile.birth.location.timezone
    ? formatInstant(profile.birth.instant, profile.birth.location.timezone)
    : ''
  return i18n.global.t('profiles.summary', {
    date: canonicalTime || profile.birth.date || '',
    time: canonicalTime ? '' : profile.birth.time || '',
    gender,
    place,
  })
}
