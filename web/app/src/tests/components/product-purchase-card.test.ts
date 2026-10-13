import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ProductPurchaseCard from '@/components/commerce/ProductPurchaseCard.vue'
import type { ChatProduct } from '@/api/commerce'

const product: ChatProduct = {
  id: 'liki-chat',
  name: 'Liki Chat',
  description: 'Liki Chat.',
  skills: ['liki'],
  display_currency: 'USD',
  display_amount_minor: 2990,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 7900 } },
    { billing_mode: 'one_time', price: { currency_code: 'USD', minor_units: 2990 } },
    { billing_mode: 'subscription', price: { currency_code: 'USD', minor_units: 1990 } },
  ],
  purchased: false,
}

describe('ProductPurchaseCard', () => {
  it('renders subscription first and emits its billing mode', async () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, { props: { product, buying: false } })
    expect(wrapper.text()).toContain('Liki Chat')
    const text = wrapper.text()
    expect(text).toContain('$19.90')
    expect(text).toContain('$29.90')
    expect(text).toContain('每月自动续费，可随时取消')

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('purchase')).toHaveLength(1)
    expect(wrapper.emitted('purchase')?.[0]).toEqual(['subscription'])
  })

  it('disables purchase for an active product', () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, {
      props: { product: { ...product, purchased: true }, buying: false },
    })
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
