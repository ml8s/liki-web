import { i18n } from '@/i18n'

type AdminStatusTagType = 'success' | 'danger' | 'info' | 'warning'

/** Records that carry a lifecycle status; wording and color are owned per domain. */
export type AdminStatusDomain =
  | 'order'
  | 'subscription'
  | 'payout'
  | 'application'
  | 'partner'
  | 'entitlement'
  | 'finance_task'
  | 'administrator'

export interface AdminStatusMeta {
  label: string
  tag: AdminStatusTagType
}

export interface AdminStatusOption {
  value: string
  label: string
}

/**
 * One dictionary per domain: the same code (paid, pending) means something
 * different to an operator per record type. Filter options and decode
 * whitelists derive from these maps so wording cannot drift per view.
 */
const adminStatusDomains: Record<AdminStatusDomain, Record<string, AdminStatusMeta>> = {
  order: {
    pending: { label: '待支付', tag: 'warning' },
    paid: { label: '已支付', tag: 'success' },
    canceled: { label: '已取消', tag: 'info' },
  },
  subscription: {
    pending: { label: '待激活', tag: 'warning' },
    active: { label: '有效', tag: 'success' },
    on_hold: { label: '暂停', tag: 'warning' },
    cancelled: { label: '已取消', tag: 'info' },
    expired: { label: '已过期', tag: 'info' },
    failed: { label: '失败', tag: 'danger' },
  },
  payout: {
    requested: { label: '待审核', tag: 'warning' },
    pending: { label: '处理中', tag: 'warning' },
    approved: { label: '已批准', tag: 'success' },
    rejected: { label: '已拒绝', tag: 'danger' },
    paid: { label: '已打款', tag: 'success' },
    failed: { label: '失败', tag: 'danger' },
  },
  application: {
    pending: { label: '待审核', tag: 'warning' },
    approved: { label: '已通过', tag: 'success' },
    rejected: { label: '已拒绝', tag: 'danger' },
  },
  partner: {
    active: { label: '有效', tag: 'success' },
    frozen: { label: '已冻结', tag: 'danger' },
  },
  entitlement: {
    active: { label: '有效', tag: 'success' },
    expired: { label: '已过期', tag: 'info' },
    revoked: { label: '已撤销', tag: 'info' },
  },
  finance_task: {
    pending: { label: '待处理', tag: 'warning' },
    resolved: { label: '已追回', tag: 'success' },
    rejected: { label: '已豁免', tag: 'info' },
  },
  administrator: {
    active: { label: '有效', tag: 'success' },
    disabled: { label: '已禁用', tag: 'danger' },
  },
}

const adminStatusAllOption: AdminStatusOption = { value: '', label: '全部' }

function statusMetaFor(domain: AdminStatusDomain): Record<string, AdminStatusMeta> {
  return adminStatusDomains[domain] ?? {}
}

/** Unknown codes stay visible as-is so operators can report them upstream. */
export function adminStatusMeta(domain: AdminStatusDomain, status: string): AdminStatusMeta {
  return statusMetaFor(domain)[status] ?? { label: status, tag: 'info' }
}

/** Filter controls render these; "全部" always closes the list. */
export function adminStatusOptions(domain: AdminStatusDomain): AdminStatusOption[] {
  return [
    ...Object.entries(statusMetaFor(domain)).map(([value, meta]) => ({ value, label: meta.label })),
    adminStatusAllOption,
  ]
}

/** The matching decode whitelist, so a filter and its URL values cannot diverge. */
export function adminStatusValues(domain: AdminStatusDomain): string[] {
  return adminStatusOptions(domain).map(option => option.value)
}

export function capabilityLabel(capability: string): string {
  const labels: Record<string, string> = {
    'entitlement.read': '权益查看',
    'entitlement.manage': '权益管理',
    'partner.read': '合作伙伴查看',
    'partner.approve': '合作伙伴审批',
    'partner.freeze': '合作伙伴冻结',
    'partner.unfreeze': '合作伙伴解冻',
  'commission.read': '佣金查看',
  'commission.void': '佣金作废',
  'sales.read': '订单查看',
    'payout.read': '提现查看',
    'payout.approve': '提现批准',
    'payout.reject': '提现拒绝',
    'payout.mark_paid': '提现打款',
    'audit.read': '审计查看',
    'admin.user.manage': '管理员管理',
    'admin.super': '超级管理员',
  }
  return labels[capability] ?? capability
}

export function adminRoleLabel(role: string): string {
  return role === 'kol' ? 'KOL' : role === 'scout' ? 'Scout' : role
}

export function adminPaymentMethodLabel(method: string): string {
  return method === 'alipay' ? 'Alipay' : method === 'paypal' ? 'PayPal' : method
}

export function financeTaskTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    commission_refund_clawback: '佣金退款追回',
    user_erasure_review: '注销后待结算',
  }
  return labels[type] ?? type
}

export function formatAdminMoney(amountMinor: number, currency: string): string {
  const amount = Number(amountMinor || 0) / 100
  if (!currency) return amount.toFixed(2)
  return new Intl.NumberFormat(i18n.global.locale.value, {
    style: 'currency',
    currency,
    currencyDisplay: 'code',
  }).format(amount).replace(/\u00a0/g, ' ')
}

export function maskAdminAccount(value: string): string {
  const account = String(value || '')
  const at = account.indexOf('@')
  if (at > 0) return `${account.slice(0, 1)}***${account.slice(at)}`
  if (account.length > 7) return `${account.slice(0, 2)}****${account.slice(-4)}`
  return '****'
}

