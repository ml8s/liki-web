import { describe, expect, it } from 'vitest'
import {
  adminStatusTagType,
  capabilityLabel,
  financeTaskTypeLabel,
  formatAdminMoney,
  maskAdminAccount,
  toDatetimeLocal,
} from '@/utils/adminFormat'

describe('admin formatting', () => {
  it('maps lifecycle states to tag types', () => {
    expect(adminStatusTagType('active')).toBe('success')
    expect(adminStatusTagType('rejected')).toBe('danger')
    expect(adminStatusTagType('requested')).toBe('warning')
    expect(adminStatusTagType('unknown')).toBe('info')
  })

  it('uses stable labels and money formatting', () => {
    expect(capabilityLabel('admin.super')).toBe('超级管理员')
    expect(capabilityLabel('custom.new')).toBe('custom.new')
    expect(financeTaskTypeLabel('user_erasure_review')).toBe('注销后待结算')
    expect(formatAdminMoney(123456, 'CNY')).toBe('CNY 1234.56')
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
