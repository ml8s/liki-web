import { request } from './http'

export type UserStatus = 'active' | 'disabled' | 'deleted'

export interface Me {
  id: string
  email: string
  status: UserStatus
  locale: string
  last_login_at?: string
  /** 有关系的合作角色（含审核中/被拒）；App 壳据此显示"我的合作"入口。 */
  partner_roles?: string[]
}

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
