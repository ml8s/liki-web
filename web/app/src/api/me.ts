import { request } from './http'
import type { Me } from '@/types'

export function fetchMe(): Promise<Me> {
  return request<Me>('/api/me')
}

export interface Entitlements {
  allowed_skills: string[]
}

export function fetchEntitlements(): Promise<Entitlements> {
  return request<Entitlements>('/api/me/entitlements')
}
