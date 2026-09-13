import { post } from './http'
import type { EmailCodeResponse, Me } from '@/types'

export function requestEmailCode(email: string): Promise<EmailCodeResponse> {
  return post<EmailCodeResponse>('/api/auth/email/code', { email })
}

export function verifyEmailCode(email: string, code: string): Promise<{ user: Me; session_id: string; expires_at: string }> {
  return post('/api/auth/email/verify', { email, code })
}

export function logout(): Promise<{ status: string }> {
  return post('/api/auth/logout')
}
