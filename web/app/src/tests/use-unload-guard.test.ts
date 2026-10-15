import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { useUnloadGuard } from '@/composables/useUnloadGuard'

/* eslint-disable vue/one-component-per-file -- each test needs a minimal host */
function createUnloadEvent() {
  return new Event('beforeunload', { cancelable: true })
}

describe('useUnloadGuard', () => {
  it('does not interrupt unload when there is no unsaved work', () => {
    const enabled = ref(false)
    mount(defineComponent({ setup: () => useUnloadGuard(enabled) }))

    const event = createUnloadEvent()
    window.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(false)
  })

  it('protects unsaved work and releases the guard on unmount', () => {
    const enabled = ref(true)
    const wrapper = mount(defineComponent({ setup: () => useUnloadGuard(enabled) }))

    const event = createUnloadEvent()
    window.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)

    wrapper.unmount()
    const eventAfterUnmount = createUnloadEvent()
    window.dispatchEvent(eventAfterUnmount)
    expect(eventAfterUnmount.defaultPrevented).toBe(false)
  })
})
