import { expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import SkillCard from '@/components/chat/SkillCard.vue'

it('renders the skill action as a working route link', async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/orders', component: { template: '<div />' } },
    ],
  })
  await router.push('/')

  const wrapper = mount(SkillCard, {
    props: {
      label: 'Liki Chat',
      description: 'Unlock the consultation.',
      enabled: false,
      actionTo: '/orders',
    },
    global: { plugins: [router] },
  })

  const link = wrapper.find('a')
  expect(link.attributes('href')).toBe('/orders')
  expect(link.find('button').exists()).toBe(false)
})
