import { get, patch, post } from './http'

export type AdminCapability =
  | 'entitlement.read'
  | 'entitlement.manage'
  | 'partner.read'
  | 'partner.approve'
  | 'partner.freeze'
  | 'partner.unfreeze'
  | 'commission.read'
  | 'commission.void'
  | 'payout.read'
  | 'payout.approve'
  | 'payout.reject'
  | 'payout.mark_paid'
  | 'audit.read'
  | 'admin.user.manage'
  | 'admin.super'

export interface AdminSession {
  email: string
  capabilities: AdminCapability[]
}

export interface AdminEntitlement {
  id: string
  user_id: string
  email: string
  product_id: string
  skills: string[]
  status: 'active' | 'revoked' | 'expired'
  source: string
  source_id: string
  starts_at: string
  expires_at: string
  created_at: string
  updated_at: string
}

export interface AdminCapabilityRecord {
  capability: AdminCapability
  status: 'active' | 'revoked'
  granted_by: string
  granted_at: string
  revoked_at?: string
}

export interface AdminAccount {
  id: string
  email: string
  user_id?: string
  status: 'active' | 'disabled'
  created_at: string
  updated_at: string
  capabilities: AdminCapabilityRecord[]
}

export type PartnerRole = 'kol' | 'scout'
export type PartnerStatus = 'active' | 'frozen'

export interface AdminApplication {
  id: string
  user_id: string
  type: PartnerRole
  code: string
  status: 'pending' | 'approved' | 'rejected'
  note: string
  reviewer_id?: string
  decision_reason?: string
  created_at: string
  reviewed_at?: string
}

export interface AdminPartner {
  user_id: string
  code: string
  role: PartnerRole
  status: PartnerStatus
  recruiter_id?: string
  created_at: string
  updated_at: string
}

export interface AdminPayout {
  id: string
  referrer_id: string
  role: PartnerRole
  amount_minor: number
  requested_amount_minor: number
  currency: string
  method: 'alipay' | 'paypal'
  account: string
  status: 'requested' | 'pending' | 'approved' | 'rejected' | 'paid' | 'failed'
  payment_reference?: string
  rejection_reason?: string
  created_at: string
  updated_at: string
  paid_at?: string
}

export interface AdminAction {
  id: string
  actor: string
  action: string
  target_type: string
  target_id: string
  before?: string
  after?: string
  reason?: string
  created_at: string
}

export interface AdminFinanceTask {
  id: string
  type: 'commission_refund_clawback' | 'user_erasure_review'
  provider: string
  payment_intent_id: string
  provider_refund_id: string
  entry_id: string
  beneficiary_id: string
  beneficiary_role: PartnerRole
  amount_minor: number
  currency: string
  status: 'pending' | 'resolved' | 'rejected'
  assignee?: string
  resolution_note?: string
  created_at: string
  updated_at: string
  resolved_at?: string
}

export interface AdminMoney {
  currency: string
  amount_minor: number
}

export interface AdminFinanceReport {
  period_start: string
  period_end: string
  one_time_paid: AdminMoney[]
  subscription_paid: AdminMoney[]
  subscription_refunded: AdminMoney[]
  net_product_revenue: AdminMoney[]
  partner_commission_net: AdminMoney[]
  partner_payout_paid: AdminMoney[]
  partner_pending_payout: AdminMoney[]
}

function query(params: Record<string, string | number | undefined>) {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') search.set(key, String(value))
  }
  const encoded = search.toString()
  return encoded ? `?${encoded}` : ''
}

export function fetchAdminSession(): Promise<AdminSession> {
  return get('/api/admin/session')
}

export function fetchAdminEntitlements(email: string, status = ''): Promise<{ entitlements: AdminEntitlement[] }> {
  return get(`/api/admin/entitlements${query({ email, status })}`)
}

export function expireAdminEntitlement(id: string, expires_at: string, reason: string): Promise<{ result: string }> {
  return patch(`/api/admin/entitlements/${encodeURIComponent(id)}/expire`, { expires_at, reason })
}

export function revokeAdminEntitlement(id: string, reason: string): Promise<{ result: string }> {
  return post(`/api/admin/entitlements/${encodeURIComponent(id)}/revoke`, { reason })
}

export function fetchAdminAccounts(): Promise<{ admins: AdminAccount[] }> {
  return get('/api/admin/admins')
}

export function createAdminAccount(email: string): Promise<{ result: string }> {
  return post('/api/admin/admins', { email })
}

export function grantAdminCapability(email: string, capability: AdminCapability, reason: string): Promise<{ result: string }> {
  return post(`/api/admin/admins/${encodeURIComponent(email)}/capabilities`, { capability, reason })
}

export function revokeAdminCapability(email: string, capability: AdminCapability, reason: string): Promise<{ result: string }> {
  return post(`/api/admin/admins/${encodeURIComponent(email)}/capabilities/revoke`, { capability, reason })
}

export function setAdminAccountStatus(email: string, status: 'active' | 'disabled', reason: string): Promise<{ result: string }> {
  return post(`/api/admin/admins/${encodeURIComponent(email)}/${status === 'active' ? 'enable' : 'disable'}`, {
    status,
    reason,
  })
}

export function fetchAdminApplications(status = 'pending', role = ''): Promise<{ applications: AdminApplication[] }> {
  return get(`/api/admin/partner-applications${query({ status, role, limit: 100 })}`)
}

export function decideAdminApplication(id: string, approve: boolean, reason: string): Promise<{ result: string }> {
  return post(`/api/admin/partner-applications/${encodeURIComponent(id)}/decision`, { approve, reason })
}

export function fetchAdminPartners(role = '', status = '', q = ''): Promise<{ partners: AdminPartner[] }> {
  return get(`/api/admin/partners${query({ role, status, q, limit: 100 })}`)
}

export function setAdminPartnerStatus(userId: string, freeze: boolean, reason: string): Promise<{ result: string }> {
  return post(`/api/admin/partners/${encodeURIComponent(userId)}/${freeze ? 'freeze' : 'unfreeze'}`, { reason })
}

export function fetchAdminPayouts(status = 'requested'): Promise<{ payouts: AdminPayout[] }> {
  return get(`/api/admin/partner-payouts${query({ status, limit: 100 })}`)
}

export function actOnAdminPayout(id: string, action: 'approve' | 'reject' | 'mark-paid', body: Record<string, string>): Promise<{ result: string }> {
  return post(`/api/admin/partner-payouts/${encodeURIComponent(id)}/${action}`, body)
}

export function revealAdminPayoutAccount(id: string, reason: string): Promise<{ account: string }> {
  return post(`/api/admin/partner-payouts/${encodeURIComponent(id)}/reveal-account`, { reason })
}

export function fetchAdminActions(): Promise<{ actions: AdminAction[] }> {
  return get('/api/admin/partner-audit-actions?limit=100')
}

export function fetchAdminFinanceTasks(status = 'pending'): Promise<{ tasks: AdminFinanceTask[] }> {
  return get(`/api/admin/partner-finance-tasks${query({ status, limit: 100 })}`)
}

export function resolveAdminFinanceTask(id: string, status: 'resolved' | 'rejected', note: string): Promise<{ result: string }> {
  return post(`/api/admin/partner-finance-tasks/${encodeURIComponent(id)}/resolve`, { status, note })
}

export function fetchAdminFinanceReport(days: 7 | 30 | 90): Promise<AdminFinanceReport> {
  return get(`/api/admin/finance/summary?days=${days}`)
}

export function compensateAdminEntitlement(
  email: string,
  template: 'support_7d' | 'support_30d',
  reason: string,
  idempotencyKey: string,
): Promise<{ template: string; entitlement: AdminEntitlement; email_queued: boolean }> {
  return post('/api/admin/entitlements/compensation', {
    email, template, reason, idempotency_key: idempotencyKey,
  })
}
