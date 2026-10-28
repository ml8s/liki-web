import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import MeView from '@/views/MeView.vue'
import { useAuthStore } from '@/stores/auth'

async function createTestRouter() {
  const router = createRouter({
    history: createMemoryHistory('/app/'),
    routes: [
      { path: '/me', name: 'app-me', component: { template: '<div />' } },
      { path: '/', name: 'app-consultation', component: { template: '<div />' } },
      { path: '/classic/birth-profiles', name: 'birth-profiles', component: { template: '<div />' } },
      { path: '/classic/chat', name: 'chat', component: { template: '<div />' } },
      { path: '/classic/orders', name: 'orders', component: { template: '<div />' } },
      { path: '/classic/account', name: 'account', component: { template: '<div />' } },
      { path: '/classic/partners', name: 'partner-center', component: { template: '<div />' } },
    ],
  })
  router.push('/me')
  await router.isReady()
  return router
}

describe('MeView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
  })

  it('composes existing profile, history, order, and account destinations', async () => {
    const auth = useAuthStore()
    auth.user = { id: 'user', email: 'u@example.com', status: 'active', locale: 'zh-Hans', partner_roles: [] }
    const router = await createTestRouter()
    const wrapper = mount(MeView, { global: { plugins: [router] } })
    await flushPromises()

    const labels = wrapper.findAll('.entry-name').map(node => node.text())
    expect(labels).toEqual(['档案', 'Liki Chat', '订单', '账户设置'])
    expect(wrapper.text()).not.toContain('我的合作')
    wrapper.unmount()
  })

  it('shows the existing partner center only for role-gated users and navigates by route name', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 'user',
      email: 'u@example.com',
      status: 'active',
      locale: 'zh-Hans',
      partner_roles: ['kol'],
    }
    const router = await createTestRouter()
    const push = vi.spyOn(router, 'push').mockResolvedValue(undefined)
    const wrapper = mount(MeView, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('我的合作')
    await wrapper.find('.entry-row').trigger('click')
    expect(push).toHaveBeenCalledWith({ name: 'birth-profiles' })
    wrapper.unmount()
  })
})
