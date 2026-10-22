import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PartnerPageShell from '@/components/partner/PartnerPageShell.vue'

const stubs = {
  ElSkeleton: { template: '<div data-testid="skeleton" />' },
  ElResult: { template: '<div data-testid="result"><slot name="extra" /></div>' },
  ElAlert: { template: '<div data-testid="alert" />' },
  ElButton: { template: '<button><slot /></button>' },
}

describe('PartnerPageShell', () => {
  it('shows skeleton when loading and not loaded', () => {
    const wrapper = mount(PartnerPageShell, {
      props: { title: 'T', subtitle: 'S', loading: true, loaded: false, error: '', showRefresh: false },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="skeleton"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(false)
  })

  it('shows error result when error and not loaded', () => {
    const wrapper = mount(PartnerPageShell, {
      props: { title: 'T', subtitle: 'S', loading: false, loaded: false, error: 'boom', showRefresh: false },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(true)
  })

  it('renders slot content when loaded', () => {
    const wrapper = mount(PartnerPageShell, {
      props: { title: 'T', subtitle: 'S', loading: false, loaded: true, error: '', showRefresh: false },
      slots: { default: '<p data-testid="content">hello</p>' },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="content"]').exists()).toBe(true)
  })

  it('shows partial error alert when loaded but error present', () => {
    const wrapper = mount(PartnerPageShell, {
      props: { title: 'T', subtitle: 'S', loading: false, loaded: true, error: 'partial', showRefresh: false },
      slots: { default: '<div />' },
      global: { stubs },
    })
    expect(wrapper.find('[data-testid="alert"]').exists()).toBe(true)
  })

  it('emits refresh when retry button clicked', async () => {
    const wrapper = mount(PartnerPageShell, {
      props: { title: 'T', subtitle: 'S', loading: false, loaded: false, error: 'boom', showRefresh: false },
      global: { stubs },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
})
