import type { Router, RouteLocationResolved, RouteRecordNormalized } from 'vue-router'

// Vue Router keeps lazy route components as loader functions. Invoking them
// downloads and evaluates the chunk without mounting a half-ready view.
export async function preloadRouteComponents(records: readonly RouteRecordNormalized[]): Promise<void> {
  await Promise.all(records.flatMap(record =>
    Object.values(record.components ?? {}).map(async component => {
      if (typeof component !== 'function') return
      await (component as unknown as () => Promise<unknown>)()
    }),
  ))
}

/**
 * RouterLink hrefs include the browser history base. Resolve that browser URL
 * back to the router's internal path so interaction preload cannot accidentally
 * create `/base/base/...`.
 */
export function resolveBrowserHref(router: Router, href: string): RouteLocationResolved {
  const base = router.options.history.base
  const path = base && href.startsWith(base) ? href.slice(base.length) || '/' : href
  return router.resolve(path)
}

/**
 * Small operators and data-saver sessions pay a higher cost for speculative
 * chunks. They still work normally; only background prefetch is skipped.
 */
export function idlePreloadAllowed(): boolean {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return false
  if (window.matchMedia('(max-width: 720px)').matches) return false

  const connection = (navigator as Navigator & {
    connection?: { saveData?: boolean }
  }).connection
  return !connection?.saveData
}
