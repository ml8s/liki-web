import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ProductPurchaseCard from '@/components/commerce/ProductPurchaseCard.vue'
import type { ChatProduct } from '@/api/commerce'

const product: ChatProduct = {
  id: 'liki-chat',
  name: 'Liki Chat',
  description: 'Unified metaphysics chat.',
  skills: ['liki'],
  prices: { CNY: 2990, USD: 2990 },
  purchased: false,
}

describe('ProductPurchaseCard', () => {
  it('renders prices and emits purchase', async () => {
    const wrapper = mount(ProductPurchaseCard, { props: { product, buying: false } })
    expect(wrapper.text()).toContain('Liki Chat')
    expect(wrapper.text()).toContain('¥29.90')
    expect(wrapper.text()).toContain('$29.90')

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('purchase')).toHaveLength(1)
  })

  it('disables purchase for an active product', () => {
    const wrapper = mount(ProductPurchaseCard, {
      props: { product: { ...product, purchased: true }, buying: false },
    })
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
