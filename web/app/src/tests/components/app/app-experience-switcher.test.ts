import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AppExperienceSwitcher from '@/components/app/AppExperienceSwitcher.vue'

describe('AppExperienceSwitcher', () => {
  it('links to the canonical consultation experience', () => {
    const wrapper = mount(AppExperienceSwitcher, {
      props: { target: 'consultation', variant: 'sidebar' },
      slots: { default: '领域咨询' },
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="String(to.name)"><slot /></a>',
          },
        },
      },
    })
    const link = wrapper.get('a')

    expect(link.text()).toBe('领域咨询')
    expect(link.attributes('href')).toBe('app-consultation')
    expect(link.classes()).toContain('experience-switcher--sidebar')
  })

  it('links back to the classic experience and supports the dock variant', () => {
    const wrapper = mount(AppExperienceSwitcher, {
      props: { target: 'classic', variant: 'dock' },
      slots: { default: '经典版' },
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="String(to.name)"><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.get('a').attributes('href')).toBe('dashboard')
    expect(wrapper.get('a').classes()).toContain('experience-switcher--dock')
  })
})
