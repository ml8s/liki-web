import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import {
  clearEngineResultCache,
  useEngineResult,
  type UseEngineResultResource,
} from '@shared/composables/useEngineResult'

function mountResource<TResult>(
  resource: UseEngineResultResource<TResult>,
) {
  return mount(defineComponent({
    setup() {
      return {
        status: resource.status,
        data: resource.data,
        error: resource.error,
        refreshing: resource.refreshing,
        lastUpdatedAt: resource.lastUpdatedAt,
      }
    },
    template: '<div />',
  }))
}

describe('useEngineResult', () => {
  beforeEach(() => {
    clearEngineResultCache()
    vi.clearAllMocks()
  })

  it('loads an input and reuses a fresh cache for the same canonical input', async () => {
    const load = vi.fn(async (input: { chart: string }) => `full:${input.chart}`)
    const input = ref({ chart: 'abc' })

    const firstResource = useEngineResult({ input, load, staleTime: 60_000 })
    const firstWrapper = mountResource(firstResource)
    await flushPromises()

    expect(firstWrapper.vm.data).toBe('full:abc')
    expect(load).toHaveBeenCalledTimes(1)
    firstWrapper.unmount()

    const secondResource = useEngineResult({ input, load, staleTime: 60_000 })
    const secondWrapper = mountResource(secondResource)
    await flushPromises()

    expect(secondWrapper.vm.data).toBe('full:abc')
    expect(load).toHaveBeenCalledTimes(1)
    secondWrapper.unmount()
  })

  it('reloads when canonical input changes and isolates results', async () => {
    const input = ref({ chart: 'a', year: 2025 })
    const load = vi.fn(async (value: { chart: string; year: number }) =>
      `result:${value.chart}:${value.year}`)

    const resource = useEngineResult({ input, load, staleTime: 60_000 })
    const wrapper = mountResource(resource)
    await flushPromises()

    expect(wrapper.vm.data).toBe('result:a:2025')

    input.value = { chart: 'b', year: 2026 }
    await flushPromises()
    expect(wrapper.vm.data).toBe('result:b:2026')
    expect(load).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('keeps stale data while revalidating and then updates it', async () => {
    const input = ref({ chart: 'stable' })
    let calls = 0
    const load = vi.fn(async () => {
      calls += 1
      return `result-${calls}`
    })

    const resource = useEngineResult({ input, load, staleTime: 0 })
    const wrapper = mountResource(resource)
    await flushPromises()
    expect(wrapper.vm.data).toBe('result-1')

    const refreshPromise = resource.refresh()
    expect(resource.refreshing.value).toBe(true)
    expect(wrapper.vm.data).toBe('result-1')

    await refreshPromise
    expect(wrapper.vm.data).toBe('result-2')
    expect(resource.refreshing.value).toBe(false)
    expect(load).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('de-duplicates concurrent requests for the same cache key', async () => {
    const load = vi.fn(async () => 'shared-result')
    const input = ref({ chart: 'same' })

    const resourceA = useEngineResult({ input, load, staleTime: 60_000 })
    const resourceB = useEngineResult({ input, load, staleTime: 60_000 })
    const wrapperA = mountResource(resourceA)
    const wrapperB = mountResource(resourceB)
    await flushPromises()

    expect(load).toHaveBeenCalledTimes(1)
    expect(wrapperA.vm.data).toBe('shared-result')
    expect(wrapperB.vm.data).toBe('shared-result')
    wrapperA.unmount()
    wrapperB.unmount()
  })

  it('exposes an error and supports retry after a failed load', async () => {
    const load = vi.fn()
      .mockRejectedValueOnce(new Error('engine unavailable'))
      .mockResolvedValueOnce('recovered')

    const resource = useEngineResult({
      input: ref({ chart: 'x' }),
      load,
      staleTime: 60_000,
    })
    const wrapper = mountResource(resource)
    await flushPromises()

    expect(resource.status.value).toBe('error')
    expect(resource.error.value?.message).toBe('engine unavailable')
    expect(resource.data.value).toBeNull()

    await resource.reload()
    expect(resource.status.value).toBe('success')
    expect(resource.data.value).toBe('recovered')
    wrapper.unmount()
  })

  it('invalidates a current result and forces a fresh load', async () => {
    const load = vi.fn(async ({ revision }: { revision: number }) => `result-${revision}`)
    const input = ref({ chart: 'abc', revision: 1 })

    const resource = useEngineResult({ input, load, staleTime: 60_000 })
    const wrapper = mountResource(resource)
    await flushPromises()
    expect(wrapper.vm.data).toBe('result-1')

    input.value = { chart: 'abc', revision: 2 }
    await flushPromises()
    expect(wrapper.vm.data).toBe('result-2')

    resource.invalidate()
    await flushPromises()
    expect(load).toHaveBeenCalledTimes(3)
    wrapper.unmount()
  })

  it('stays idle for invalid or disabled input', async () => {
    const load = vi.fn(async () => 'should-not-load')
    const enabled = ref(false)
    const input = ref<Record<string, string> | null>(null)

    const resource = useEngineResult({
      input,
      load,
      enabled,
      staleTime: 60_000,
    })
    const wrapper = mountResource(resource)
    await flushPromises()

    expect(resource.status.value).toBe('idle')
    expect(load).not.toHaveBeenCalled()

    enabled.value = true
    await nextTick()
    await flushPromises()
    expect(resource.status.value).toBe('idle')
    expect(load).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
