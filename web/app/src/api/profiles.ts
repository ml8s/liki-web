import { get, post, request } from './http'
import { i18n } from '@/i18n'

export interface Profile {
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

export interface ProfileInput {
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

export interface ProfileListResult {
  profiles: Profile[]
  next_cursor: string
}

export function fetchProfiles(cursor?: string): Promise<ProfileListResult> {
  const query = new URLSearchParams({ limit: '50' })
  if (cursor) query.set('cursor', cursor)
  return get<ProfileListResult>(`/api/profiles?${query.toString()}`)
}

export function fetchProfile(id: string): Promise<Profile> {
  return get<Profile>(`/api/profiles/${id}`)
}

export function createProfile(input: ProfileInput): Promise<Profile> {
  return post<Profile>('/api/profiles', input)
}

export function updateProfile(id: string, expectedVersion: number, input: ProfileInput): Promise<Profile> {
  return request<Profile>(`/api/profiles/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ expected_version: expectedVersion, ...input }),
  })
}

export function deleteProfile(id: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/profiles/${id}`, { method: 'DELETE' })
}

export function birthSummary(profile: Profile): string {
  const t = i18n.global.t
  const gender = profile.gender === 'male' ? t('profiles.male') : t('profiles.female')
  const calendar = profile.calendar === 'lunar' ? t('profiles.form.lunar') : t('profiles.form.solar')
  const known = {
    known: t('chat.systemProfile.known'),
    approximate: t('chat.systemProfile.approximate'),
    unknown: t('chat.systemProfile.unknown'),
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
