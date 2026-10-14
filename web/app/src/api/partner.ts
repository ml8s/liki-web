import { get, post } from './http'

export type PartnerRole = 'kol' | 'scout'
export type PartnerStatus = 'active' | 'frozen'
export type PartnerRoleStatus = 'active' | 'frozen' | 'retired'
export type CommissionStatus = 'pending' | 'paid' | 'voided'
export type PayoutStatus = 'requested' | 'pending' | 'approved' | 'rejected' | 'paid' | 'failed'
export type PayoutMethod = 'alipay' | 'paypal'
export type ApplicationStatus = 'pending' | 'approved' | 'rejected'

export interface Partner {
  user_id: string
  code: string
  role: PartnerRole
  status: PartnerStatus
  recruiter_id?: string
  created_at: string
  updated_at: string
}

export interface PartnerApplication {
  id: string
  user_id: string
  type: PartnerRole
  code: string
  status: ApplicationStatus
  note: string
  reviewer_id?: string
  decision_reason?: string
  created_at: string
  reviewed_at?: string
}

export interface PartnerProgramState {
  role: PartnerRole
  status?: PartnerRoleStatus
  application: PartnerApplication | null
  can_submit: boolean
}

export interface PartnerStates {
  kol: PartnerProgramState
  scout: PartnerProgramState
}

export interface BalanceSummary {
  currency: string
  total_minor: number
  pending_minor: number
  paid_minor: number
}

export interface KOLOverview {
  partner: Partner
  attributed_users: number
  balances: BalanceSummary[]
}

export interface KOLSession {
  email: string
  partner: Partner
  overview: KOLOverview
}

export interface ScoutRecruit {
  user_id: string
  code: string
  created_at: string
  order_count: number
  commission_balances: Array<{ currency: string; amount_minor: number }>
}

export interface ScoutBalanceSummary {
  currency: string
  sales_minor: number
  commission_minor: number
  pending_minor: number
  paid_minor: number
}

export interface ScoutOverview {
  partner: Partner
  lead_count: number
  recruit_count: number
  order_count: number
  balances: ScoutBalanceSummary[]
  recruits: ScoutRecruit[]
}

export interface ScoutSession {
  email: string
  overview: ScoutOverview
}

export interface Settlement {
  role: PartnerRole
  status: PartnerRoleStatus
  balances: BalanceSummary[]
  payouts: Payout[]
}

export interface PartnerCommission {
  id: string
  payment_provider: string
  payment_intent_id: string
  business_order_id: string
  product: string
  buyer_user_id: string
  referrer_id: string
  recruiter_id?: string
  amount_minor: number
  currency: string
  referrer_rate_bps: number
  referrer_commission_minor: number
  recruiter_rate_bps?: number
  recruiter_commission_minor?: number
  status: CommissionStatus
  refunded_minor: number
  created_at: string
  updated_at: string
  paid_at?: string
}

export interface Payout {
  id: string
  referrer_id: string
  role: PartnerRole
  amount_minor: number
  requested_amount_minor: number
  currency: string
  method: PayoutMethod
  account: string
  status: PayoutStatus
  payment_reference?: string
  rejection_reason?: string
  created_at: string
  updated_at: string
  paid_at?: string
}

export interface ScoutSale {
  order_ref: string
  product: string
  buyer_ref: string
  amount_minor: number
  currency: string
  scout_commission_minor: number
  status: CommissionStatus
  refunded_minor: number
  created_at: string
}

export function fetchPartnerStates(): Promise<PartnerStates> {
  return get('/api/partner-portal')
}

export function fetchKOLSession(): Promise<KOLSession> {
  return get('/api/partner-portal/kol/session')
}

export function fetchKOLCommissions(): Promise<{
  commissions: PartnerCommission[]
  balances: BalanceSummary[]
}> {
  return get('/api/partner-portal/kol/commissions')
}

export function fetchKOLPayouts(): Promise<{ payouts: Payout[] }> {
  return get('/api/partner-portal/kol/payouts')
}

export function fetchKOLSettlement(): Promise<Settlement> {
  return get('/api/partner-portal/kol/settlement')
}

export function createKOLPayout(input: { method: PayoutMethod; account: string }): Promise<{ payout: Payout }> {
  return post('/api/partner-portal/kol/payouts', input)
}

export function submitApplication(role: PartnerRole, note: string): Promise<{ application: PartnerApplication }> {
  return post(`/api/${role}/application`, { note })
}

export function fetchScoutSession(): Promise<ScoutSession> {
  return get('/api/partner-portal/scout/session')
}

export function fetchScoutRecruits(): Promise<{ recruits: ScoutRecruit[] }> {
  return get('/api/partner-portal/scout/recruits')
}

export function fetchScoutCommissions(): Promise<{
  sales: ScoutSale[]
  balances: Array<{ currency: string; amount_minor: number }>
}> {
  return get('/api/partner-portal/scout/commissions')
}

export function fetchScoutPayouts(): Promise<{ payouts: Payout[] }> {
  return get('/api/partner-portal/scout/payouts')
}

export function fetchScoutSettlement(): Promise<Settlement> {
  return get('/api/partner-portal/scout/settlement')
}

export function createScoutPayout(input: { method: PayoutMethod; account: string }): Promise<{ payout: Payout }> {
  return post('/api/partner-portal/scout/payouts', input)
}
