import { describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { useNearBottom } from '@/composables/useNearBottom'

describe('useNearBottom', () => {
  it('measures one scroll element and refreshes on scroll', async () => {
    const element = document.createElement('div')
    let exposed!: ReturnType<typeof useNearBottom>
    const TestComponent = defineComponent({
      setup() {
        exposed = useNearBottom(() => element)
        return () => '<div />'
      },
    })
    const wrapper = mount(TestComponent)
    Object.defineProperties(element, {
      scrollHeight: { configurable: true, value: 1_000 },
      clientHeight: { configurable: true, value: 400 },
      scrollTop: { configurable: true, value: 0, writable: true },
    })

    exposed.measure()
    expect(exposed.nearBottom.value).toBe(false)

    element.scrollTop = 650
    element.dispatchEvent(new Event('scroll'))
    await wrapper.vm.$nextTick()
    expect(exposed.nearBottom.value).toBe(true)

    wrapper.unmount()
  })
})
