import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import { preloadRouteComponents, preloadWhenIdle } from '@/utils/routes'

function route(components: Record<string, unknown>): RouteRecordNormalized {
  return { components } as unknown as RouteRecordNormalized
}

describe('route preloading', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads lazy route chunks and ignores mounted components', async () => {
    const lazy = vi.fn().mockResolvedValue(undefined)
    await preloadRouteComponents([
      route({ default: lazy, other: {} }),
      route({}),
    ])
    expect(lazy).toHaveBeenCalledTimes(1)
  })

  it('defers preload until idle when the API is unavailable', async () => {
    vi.useFakeTimers()
    const task = vi.fn().mockResolvedValue(undefined)
    preloadWhenIdle(task, 2000)
    expect(task).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1000)
    await vi.runAllTimersAsync()
    expect(task).toHaveBeenCalledTimes(1)
  })
})
