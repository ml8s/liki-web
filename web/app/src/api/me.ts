import { request } from './http'
import type { Me } from '@/types'

export interface SessionInfo {
  id: string
  created_at: string
  expires_at: string
  last_seen_at: string
  ip_hash: string
  user_agent: string
  is_current: boolean
}

export function fetchSessions(): Promise<SessionInfo[]> {
  return request<SessionInfo[]>('/api/me/sessions')
}

export function revokeSession(sessionId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/me/sessions/${sessionId}`, { method: 'DELETE' })
}

export function revokeAllSessions(): Promise<{ status: string; count: number }> {
  return request<{ status: string; count: number }>('/api/auth/sessions/revoke-all', { method: 'POST' })
}

export interface Entitlement {
  id: string
  user_id: string
  product_id: string
  skills: string[]
  status: 'active' | 'revoked' | 'expired'
  starts_at: string
  expires_at: string
}

export interface EntitlementSummary {
  entitlements: Entitlement[]
  allowed_skills: string[]
}

export function fetchMe(): Promise<Me> {
  return request<Me>('/api/me')
}

export function fetchEntitlements(): Promise<EntitlementSummary> {
  return request<EntitlementSummary>('/api/me/entitlements')
}
