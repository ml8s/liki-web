import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import { createMemoryHistory, createRouter } from 'vue-router'
import { preloadRouteComponents, resolveBrowserHref } from '@/utils/routes'

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

describe('browser href resolution', () => {
  it('strips the history base before resolving link targets', () => {
    const router = createRouter({
      history: createMemoryHistory('/app/'),
      routes: [{ path: '/admin', name: 'admin-overview', component: {} }],
    })

    expect(resolveBrowserHref(router, '/app/admin').name).toBe('admin-overview')
  })
})
