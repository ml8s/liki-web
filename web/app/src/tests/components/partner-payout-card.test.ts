import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PartnerPayoutCard from '@/components/partner/PartnerPayoutCard.vue'
import type { Payout } from '@/api/partner'

const payouts: Payout[] = [
  {
    id: 'p1',
    beneficiary_id: 'u1', beneficiary_code: 'CODE',
    role: 'kol',
    amount_minor: 10000,
    requested_amount_minor: 10000,
    currency: 'CNY',
    method: 'alipay',
    account: 'user@test',
    status: 'paid',
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
  },
]

const stubs = {
  ElCard: { template: '<div><slot name="header" /><slot /></div>' },
  ElAlert: { template: '<div data-testid="rules" />' },
  ElTable: { template: '<div data-testid="table" />' },
  ElTag: { template: '<span data-testid="tag" />' },
  ElEmpty: { template: '<div data-testid="empty" />' },
  ElButton: { template: '<button><slot /></button>' },
  ElDialog: { template: '<div data-testid="dialog"><slot /></div>' },
}

describe('PartnerPayoutCard', () => {
  it('renders table on desktop with payouts', () => {
    const wrapper = mount(PartnerPayoutCard, {
      props: { payouts, isMobile: false, submitting: false, roleName: 'KOL' },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="table"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="empty"]').exists()).toBe(false)
  })

  it('renders record cards on mobile', () => {
    const wrapper = mount(PartnerPayoutCard, {
      props: { payouts, isMobile: true, submitting: false, roleName: 'KOL' },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="table"]').exists()).toBe(false)
    expect(wrapper.find('.record-card').exists()).toBe(true)
  })

  it('renders empty state when no payouts', () => {
    const wrapper = mount(PartnerPayoutCard, {
      props: { payouts: [], isMobile: false, submitting: false, roleName: 'KOL' },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="empty"]').exists()).toBe(true)
  })
})
