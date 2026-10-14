import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import { idlePreloadAllowed, preloadRouteComponents, preloadWhenIdle } from '@/utils/routes'

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

describe('idle preload budget', () => {
  it('skips speculative chunks on small operators', () => {
    vi.stubGlobal('innerWidth', 390)
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: true }))
    expect(idlePreloadAllowed()).toBe(false)
  })

  it('allows desktop chunks unless save-data is enabled', () => {
    vi.stubGlobal('innerWidth', 1440)
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))
    vi.stubGlobal('navigator', { connection: { saveData: false } })
    expect(idlePreloadAllowed()).toBe(true)
    vi.stubGlobal('navigator', { connection: { saveData: true } })
    expect(idlePreloadAllowed()).toBe(false)
  })
})
