import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AppLiveRegion from '@/components/app/AppLiveRegion.vue'

describe('AppLiveRegion', () => {
  it('uses polite status semantics by default', () => {
    const wrapper = mount(AppLiveRegion, {
      slots: { default: '正在生成解读…' },
    })

    const region = wrapper.find('div')
    expect(region.attributes('role')).toBe('status')
    expect(region.attributes('aria-live')).toBe('polite')
    expect(region.attributes('aria-atomic')).toBe('true')
    expect(region.text()).toBe('正在生成解读…')
    expect(region.classes()).not.toContain('app-live-region--screen-only')
    wrapper.unmount()
  })

  it('supports assertive errors and screen-only announcements', () => {
    const wrapper = mount(AppLiveRegion, {
      props: { level: 'assertive', role: 'alert', screenOnly: true },
      slots: { default: '发送失败' },
    })

    const region = wrapper.find('div')
    expect(region.attributes('role')).toBe('alert')
    expect(region.attributes('aria-live')).toBe('assertive')
    expect(region.classes()).toContain('app-live-region--screen-only')
    wrapper.unmount()
  })
})
