import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import { createMemoryHistory, createRouter } from 'vue-router'
import { idlePreloadAllowed, preloadRouteComponents, resolveBrowserHref } from '@/utils/routes'

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

describe('browser href resolution', () => {
  it('strips the history base before resolving link targets', () => {
    const router = createRouter({
      history: createMemoryHistory('/app/'),
      routes: [{ path: '/admin', name: 'admin-overview', component: {} }],
    })

    expect(resolveBrowserHref(router, '/app/admin').name).toBe('admin-overview')
  })
})
