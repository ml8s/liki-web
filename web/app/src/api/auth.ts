import { post } from './http'
import type { Me } from './me'

export interface EmailCodeResponse {
  email: string
  expires_at: string
  dev_code?: string
}

export function requestEmailCode(email: string, locale: string): Promise<EmailCodeResponse> {
  return post<EmailCodeResponse>('/api/auth/email/code', { email, locale })
}

export function verifyEmailCode(email: string, code: string, locale: string): Promise<{ user: Me; session_id: string; expires_at: string }> {
  return post('/api/auth/email/verify', { email, code, locale })
}

export function logout(): Promise<{ status: string }> {
  return post('/api/auth/logout')
}
