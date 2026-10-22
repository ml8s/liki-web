import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PartnerReferralCard from '@/components/partner/PartnerReferralCard.vue'

const stubs = {
  ElCard: { template: '<div><slot name="header" /><slot /></div>' },
  ElInput: { template: '<input readonly />' },
  ElButton: { template: '<button><slot /></button>' },
}

describe('PartnerReferralCard', () => {
  it('renders referral link input as readonly', () => {
    const wrapper = mount(PartnerReferralCard, {
      props: {
        title: 'Invite Link',
        referralLink: 'https://liki.app/r/abc',
        inputName: 'scout-referral-link',
        stats: [{ key: 'k', label: 'Leads', value: 3 }],
      },
      global: { stubs },
    })
    const input = wrapper.find('input[name="scout-referral-link"]')
    expect(input.exists()).toBe(true)
    expect(input.attributes('readonly')).toBeDefined()
  })

  it('emits copy when button clicked', async () => {
    const wrapper = mount(PartnerReferralCard, {
      props: {
        title: 'Invite',
        referralLink: 'https://liki.app/r/abc',
        inputName: 'link',
        stats: [],
      },
      global: { stubs },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('copy')).toHaveLength(1)
  })

  it('renders stats via PartnerStatGrid', () => {
    const wrapper = mount(PartnerReferralCard, {
      props: {
        title: 'Invite',
        referralLink: '',
        inputName: 'link',
        stats: [
          { key: 'a', label: 'A', value: 1 },
          { key: 'b', label: 'B', value: 2 },
        ],
      },
      global: { stubs },
    })
    expect(wrapper.find('.partner-stat-grid').exists()).toBe(true)
  })
})
