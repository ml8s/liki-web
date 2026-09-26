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
  prices: { CNY: 100, USD: 100 },
  display_currency: 'CNY',
  display_amount_minor: 100,
  purchased: false,
}

describe('ProductPurchaseCard', () => {
  it('renders price and emits purchase', async () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, { props: { product, buying: false } })
    expect(wrapper.text()).toContain('Liki Chat')
    const text = wrapper.text()
    const hasCny = text.includes('¥1.00')
    const hasUsd = text.includes('$1.00')
    expect(hasCny || hasUsd).toBe(true)

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('purchase')).toHaveLength(1)
  })

  it('disables purchase for an active product', () => {
    setActivePinia(createPinia())
    const wrapper = mount(ProductPurchaseCard, {
      props: { product: { ...product, purchased: true }, buying: false },
    })
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
