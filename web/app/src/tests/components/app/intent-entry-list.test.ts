import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import IntentEntryList, { type IntentEntry } from '@/components/app/IntentEntryList.vue'

const items = [
  {
    id: 'button-entry',
    glyph: '问',
    label: 'Button entry',
    desc: 'Emits select',
  },
  {
    id: 'route-entry',
    glyph: '档',
    label: 'Route entry',
    desc: 'Uses a native link',
    to: { name: 'profiles' },
  },
]

async function mountWithRouter(props: { items: IntentEntry[] }) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/profiles', name: 'profiles', component: { template: '<div />' } }],
  })
  await router.push('/profiles')
  await router.isReady()
  return mount(IntentEntryList, { props, global: { plugins: [router] } })
}

describe('IntentEntryList', () => {
  it('emits select for legacy button entries', async () => {
    const wrapper = await mountWithRouter({ items })
    await wrapper.findAll('.entry-row')[0]!.trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual(['button-entry'])
    wrapper.unmount()
  })

  it('renders route entries as native links for keyboard and assistive tech', async () => {
    const wrapper = await mountWithRouter({ items })
    const link = wrapper.findAll('.entry-row')[1]!

    expect(link.element.tagName).toBe('A')
    expect(link.text()).toContain('Route entry')
    wrapper.unmount()
  })
})
