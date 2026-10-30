import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import FlowDomainStrip from '@/components/flow/FlowDomainStrip.vue'
import { appDomains } from '@shared/domain/flow'

describe('FlowDomainStrip', () => {
  it('moves focus without activating until Enter is pressed', async () => {
    const wrapper = mount(FlowDomainStrip, {
      props: { domains: appDomains, modelValue: 'natal' },
      attachTo: document.body,
    })
    const tabs = wrapper.findAll('[role="tab"]')

    await tabs[0]!.trigger('keydown', { key: 'ArrowRight' })
    expect(document.activeElement).toBe(tabs[1]!.element)
    expect(tabs[1]!.attributes('aria-selected')).toBe('false')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await tabs[1]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['divination'])
    wrapper.unmount()
  })

  it('renders the four canonical domains in one strip', () => {
    const wrapper = mount(FlowDomainStrip, {
      props: { domains: appDomains, modelValue: 'natal' },
    })
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(4)
    wrapper.unmount()
  })
})
