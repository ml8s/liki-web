import { get, post, request } from './http'

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
  return get<ProfileListResult>(`/api/profiles?limit=50${cursor ? '&cursor=' + cursor : ''}`)
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
  const gender = profile.gender === 'male' ? '男' : '女'
  const calendar = profile.calendar === 'lunar' ? '农历' : '公历'
  const known = { known: '已知', approximate: '时辰', unknown: '不确定' }[profile.birth_time_known]
  const place = profile.birth_place ? ` · ${profile.birth_place}` : ''
  return `${calendar} ${profile.birth_date}${profile.birth_time ? ` ${profile.birth_time}` : ''} · ${gender} · ${known}${place}`
}