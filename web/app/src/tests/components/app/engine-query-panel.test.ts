import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import { clearEngineResultCache } from '@shared/composables/useEngineResult'

const resultSlot = {
  default: `<template #default="{ result }">
    <div data-testid="result">{{ result.value }}</div>
  </template>`,
}

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(EngineQueryPanel, {
    props: {
      resource: 'test.resource',
      input: { chart: 'abc' },
      load: async () => ({ value: 'ok' }),
      fallback: '加载失败',
      ...overrides,
    },
    slots: resultSlot,
  })
}

beforeEach(() => {
  clearEngineResultCache()
  vi.clearAllMocks()
})

describe('EngineQueryPanel', () => {
  it('renders the loaded result through the slot', async () => {
    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.get('[data-testid="result"]').text()).toBe('ok')
    wrapper.unmount()
  })

  it('localizes the failure by code and retries on demand', async () => {
    const load = vi.fn()
      .mockRejectedValueOnce(Object.assign(new Error('boom'), { code: 'engine_unavailable' }))
      .mockResolvedValueOnce({ value: 'recovered' })
    const wrapper = mountPanel({ load })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('排盘服务暂时不可用')

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="result"]').text()).toBe('recovered')
    expect(load).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('reloads when refreshToken changes', async () => {
    const load = vi.fn(async () => ({ value: 'ok' }))
    const wrapper = mountPanel({ load })
    await flushPromises()
    expect(load).toHaveBeenCalledTimes(1)

    await wrapper.setProps({ refreshToken: 1 })
    await flushPromises()

    expect(load).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('emits update:data when the result resolves', async () => {
    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.emitted('update:data')?.at(-1)).toEqual([{ value: 'ok' }])
    wrapper.unmount()
  })

  it('stays idle and skips the request when disabled', async () => {
    const load = vi.fn(async () => ({ value: 'ok' }))
    const wrapper = mountPanel({ load, enabled: false })
    await flushPromises()

    expect(load).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
