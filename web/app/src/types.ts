export type UserStatus = 'active' | 'disabled' | 'deleted'

export interface Me {
  id: string
  email: string
  status: UserStatus
  locale: string
  last_login_at?: string
}

export interface EmailCodeResponse {
  email: string
  expires_at: string
  dev_code?: string
}
