import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import type { Product } from '@/api/sales'

// 服务端只下发本区域可结算的选项，且币种恒等于 display_currency：境外是 Dodo 的
// 按次 + 订阅，国内是 Xunhu 的按次。卡片只渲染服务端给的结果。
const product: Product = {
  id: 'liki-chat',
  name: 'Liki Chat',
  description: 'Liki Chat.',
  skills: ['liki'],
  display_currency: 'USD',
  display_amount_minor: 2990,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'USD', minor_units: 2990 } },
    { billing_mode: 'subscription', price: { currency_code: 'USD', minor_units: 1990 } },
  ],
  purchased: false,
}

const domesticProduct: Product = {
  ...product,
  display_currency: 'CNY',
  display_amount_minor: 7900,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 7900 } },
  ],
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

  it('keeps the domestic card on the single one-time plan', async () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, { props: { product: domesticProduct, buying: false } })
    expect(wrapper.text()).toContain('¥79.00')
    expect(wrapper.text()).not.toContain('每月自动续费，可随时取消')
    expect(wrapper.findAll('button')).toHaveLength(1)

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('purchase')?.[0]).toEqual(['one_time'])
  })

  it('disables purchase for an active product', () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, {
      props: { product: { ...product, purchased: true }, buying: false },
    })
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
