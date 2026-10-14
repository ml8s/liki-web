import { i18n } from '@/i18n'

type AdminStatusTagType = 'success' | 'danger' | 'info' | 'warning'

export interface AdminStatusMeta {
  label: string
  tag: AdminStatusTagType
}

/** One map owns both operator language and status color so states cannot drift. */
export const adminStatusMeta: Record<string, AdminStatusMeta> = {
  active: { label: '有效', tag: 'success' },
  approved: { label: '已批准', tag: 'success' },
  disabled: { label: '已禁用', tag: 'danger' },
  expired: { label: '已过期', tag: 'info' },
  failed: { label: '失败', tag: 'danger' },
  frozen: { label: '已冻结', tag: 'danger' },
  paid: { label: '已打款', tag: 'success' },
  pending: { label: '处理中', tag: 'warning' },
  rejected: { label: '已拒绝', tag: 'danger' },
  requested: { label: '待审核', tag: 'warning' },
  resolved: { label: '已解决', tag: 'success' },
  revoked: { label: '已撤销', tag: 'info' },
}

export function adminStatusTagType(status: string): AdminStatusTagType {
  return adminStatusMeta[status]?.tag ?? 'info'
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

export function adminStatusLabel(status: string): string {
  return adminStatusMeta[status]?.label ?? status
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

export function toDatetimeLocal(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (number: number) => String(number).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}
