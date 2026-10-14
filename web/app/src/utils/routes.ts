import type { RouteRecordNormalized } from 'vue-router'

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

export function preloadWhenIdle(task: () => Promise<void>, timeout = 2000): void {
  if (typeof window === 'undefined') return
  const idleWindow = window as Window & {
    requestIdleCallback?: (callback: () => void, options?: { timeout?: number }) => number
  }
  if (idleWindow.requestIdleCallback) {
    idleWindow.requestIdleCallback(() => { void task() }, { timeout })
    return
  }
  window.setTimeout(() => { void task() }, Math.min(timeout, 1000))
}
