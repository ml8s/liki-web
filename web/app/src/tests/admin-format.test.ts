import { describe, expect, it } from 'vitest'
import {
  adminStatusLabel,
  adminStatusOptions,
  adminStatusTagType,
  capabilityLabel,
  financeTaskTypeLabel,
  formatAdminMoney,
  maskAdminAccount,
  toDatetimeLocal,
  type AdminStatusDomain,
} from '@/utils/adminFormat'

const ADMIN_STATUS_DOMAINS: AdminStatusDomain[] = [
  'order', 'subscription', 'payout', 'application', 'partner', 'entitlement', 'finance_task', 'administrator',
]

describe('admin formatting', () => {
  it('keeps one status code from meaning two things', () => {
    expect(adminStatusLabel('order', 'paid')).toBe('已支付')
    expect(adminStatusLabel('payout', 'paid')).toBe('已打款')
    expect(adminStatusLabel('order', 'pending')).toBe('待支付')
    expect(adminStatusLabel('subscription', 'pending')).toBe('待激活')
    expect(adminStatusLabel('payout', 'pending')).toBe('处理中')
    expect(adminStatusLabel('finance_task', 'pending')).toBe('待处理')
  })

  it('maps lifecycle states to tag types per domain', () => {
    expect(adminStatusTagType('payout', 'requested')).toBe('warning')
    expect(adminStatusTagType('application', 'approved')).toBe('success')
    expect(adminStatusTagType('partner', 'frozen')).toBe('danger')
    expect(adminStatusTagType('administrator', 'disabled')).toBe('danger')
    expect(adminStatusTagType('entitlement', 'revoked')).toBe('info')
  })

  it('labels subscription-only lifecycle states', () => {
    expect(adminStatusLabel('subscription', 'on_hold')).toBe('暂停')
    expect(adminStatusLabel('subscription', 'cancelled')).toBe('已取消')
    expect(adminStatusTagType('subscription', 'on_hold')).toBe('warning')
    expect(adminStatusTagType('subscription', 'cancelled')).toBe('info')
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
    }
  })

  it('uses stable labels and money formatting', () => {
    expect(adminStatusLabel('partner', 'frozen')).toBe('已冻结')
    expect(adminStatusLabel('application', 'approved')).toBe('已通过')
    expect(adminStatusLabel('order', 'unknown.new')).toBe('unknown.new')
    expect(adminStatusTagType('order', 'unknown.new')).toBe('info')
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

  it('normalizes timestamps for datetime inputs', () => {
    expect(toDatetimeLocal('2026-10-14T08:30:00Z')).toMatch(/^2026-10-14T\d{2}:30$/)
    expect(toDatetimeLocal('invalid')).toBe('')
    expect(toDatetimeLocal(null)).toBe('')
  })
})
