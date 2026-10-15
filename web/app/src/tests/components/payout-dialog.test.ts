import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PayoutDialog from '@/components/partner/PayoutDialog.vue'

describe('PayoutDialog', () => {
  it('submits the payout form on Enter with a trimmed account', async () => {
    const wrapper = mount(PayoutDialog, {
      props: {
        modelValue: true,
        title: '申请提现',
        submitting: false,
      },
      global: {
        stubs: {
          ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        },
      },
    })

    await wrapper.find('input[name="payout-account"]').setValue('  account@example  ')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toEqual([
      [{ method: 'alipay', account: 'account@example' }],
    ])
  })

  it('does not submit when the payout account is blank', async () => {
    const wrapper = mount(PayoutDialog, {
      props: {
        modelValue: true,
        title: '申请提现',
        submitting: false,
      },
      global: {
        stubs: {
          ElDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        },
      },
    })

    await wrapper.find('input[name="payout-account"]').setValue('   ')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})
