import { describe, expect, it } from 'vitest'
import { partnerCommissionStatus, partnerPayoutStatus } from '@/utils/partnerStatus'

describe('partner status', () => {
  it('maps commission state to the current customer label and action color', () => {
    expect(partnerCommissionStatus('pending')).toEqual({
      label: '待结算',
      tag: 'warning',
    })
    expect(partnerCommissionStatus('paid').tag).toBe('success')
    expect(partnerCommissionStatus('voided').tag).toBe('danger')
  })

  it('maps payout state to the current customer label and action color', () => {
    expect(partnerPayoutStatus('requested')).toEqual({
      label: '已申请',
      tag: 'warning',
    })
    expect(partnerPayoutStatus('paid').tag).toBe('success')
    expect(partnerPayoutStatus('failed').tag).toBe('danger')
  })
})
