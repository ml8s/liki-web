export function adminStatusTagType(status: string): 'success' | 'danger' | 'info' | 'warning' {
  if (['active', 'approved', 'paid', 'succeeded', 'resolved'].includes(status)) return 'success'
  if (['disabled', 'rejected', 'failed', 'frozen'].includes(status)) return 'danger'
  if (['pending', 'requested', 'on_hold'].includes(status)) return 'warning'
  return 'info'
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

export function financeTaskTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    commission_refund_clawback: '佣金退款追回',
    user_erasure_review: '注销后待结算',
  }
  return labels[type] ?? type
}

export function formatAdminMoney(amountMinor: number, currency: string): string {
  return `${currency || ''} ${(Number(amountMinor || 0) / 100).toFixed(2)}`.trim()
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
