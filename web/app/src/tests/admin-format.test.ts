import { describe, expect, it } from 'vitest'
import {
  adminStatusMeta,
  adminStatusOptions,
  adminStatusValues,
  capabilityLabel,
  financeTaskTypeLabel,
  formatAdminMoney,
  maskAdminAccount,
  type AdminStatusDomain,
} from '@/utils/adminFormat'

const ADMIN_STATUS_DOMAINS: AdminStatusDomain[] = [
  'order', 'subscription', 'payout', 'application', 'partner', 'entitlement', 'finance_task', 'administrator',
]

const label = (domain: AdminStatusDomain, status: string) => adminStatusMeta(domain, status).label
const tag = (domain: AdminStatusDomain, status: string) => adminStatusMeta(domain, status).tag

describe('admin formatting', () => {
  it('keeps one status code from meaning two things', () => {
    expect(label('order', 'paid')).toBe('已支付')
    expect(label('payout', 'paid')).toBe('已打款')
    expect(label('order', 'pending')).toBe('待支付')
    expect(label('subscription', 'pending')).toBe('待激活')
    expect(label('payout', 'pending')).toBe('处理中')
    expect(label('finance_task', 'pending')).toBe('待处理')
  })

  it('maps lifecycle states to tag types per domain', () => {
    expect(tag('payout', 'requested')).toBe('warning')
    expect(tag('application', 'approved')).toBe('success')
    expect(tag('partner', 'frozen')).toBe('danger')
    expect(tag('administrator', 'disabled')).toBe('danger')
    expect(tag('entitlement', 'revoked')).toBe('info')
  })

  it('labels subscription-only lifecycle states', () => {
    expect(label('subscription', 'on_hold')).toBe('暂停')
    expect(label('subscription', 'cancelled')).toBe('已取消')
    expect(tag('subscription', 'on_hold')).toBe('warning')
    expect(tag('subscription', 'cancelled')).toBe('info')
  })

  it('derives filter options and decode whitelists from one list', () => {
    const payout = adminStatusOptions('payout')
    expect(payout.map(option => option.value)).toEqual([
      'requested', 'pending', 'approved', 'rejected', 'paid', 'failed', '',
    ])
    expect(payout.find(option => option.value === 'paid')?.label).toBe('已打款')
    for (const domain of ADMIN_STATUS_DOMAINS) {
      const options = adminStatusOptions(domain)
      expect(options.at(-1)).toEqual({ value: '', label: '全部' })
      expect(new Set(options.map(option => option.value)).size).toBe(options.length)
      // The decode whitelist is the same list a filter control renders.
      expect(adminStatusValues(domain)).toEqual(options.map(option => option.value))
    }
  })

  it('uses stable labels and money formatting', () => {
    expect(label('partner', 'frozen')).toBe('已冻结')
    expect(label('application', 'approved')).toBe('已通过')
    expect(label('order', 'unknown.new')).toBe('unknown.new')
    expect(tag('order', 'unknown.new')).toBe('info')
    expect(capabilityLabel('admin.super')).toBe('超级管理员')
    expect(capabilityLabel('custom.new')).toBe('custom.new')
    expect(financeTaskTypeLabel('user_erasure_review')).toBe('注销后待结算')
    expect(formatAdminMoney(123456, 'CNY')).toBe('CNY 1,234.56')
    expect(formatAdminMoney(0, '')).toBe('0.00')
  })

  it('masks payout accounts without retaining extra data', () => {
    expect(maskAdminAccount('user@example.com')).toBe('u***@example.com')
    expect(maskAdminAccount('1234567890')).toBe('12****7890')
    expect(maskAdminAccount('12')).toBe('****')
  })
})
