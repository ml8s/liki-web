import { ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { canonicalQuery } from '@/utils/query'

export type AdminPageState = Record<string, number>

export interface AdminQueryStateOptions<TFilters extends Record<string, unknown>> {
  defaults: TFilters
  pageKeys: string[]
  decode: (query: LocationQuery) => TFilters
  encode: (filters: TFilters, pages: AdminPageState) => Record<string, string | number | undefined>
  onApplied: (
    filters: TFilters,
    pages: AdminPageState,
    source: 'initial' | 'submit' | 'route',
  ) => void | Promise<void>
}

function cloneFilters<TFilters extends Record<string, unknown>>(filters: TFilters): TFilters {
  return JSON.parse(JSON.stringify(filters)) as TFilters
}

function decodePageState(query: LocationQuery, pageKeys: string[]): AdminPageState {
  return Object.fromEntries(pageKeys.map(key => {
    const raw = query[key]
    const value = Array.isArray(raw) ? raw[0] : raw
    const parsed = Number.parseInt(String(value ?? '1'), 10)
    return [key, Number.isFinite(parsed) && parsed > 0 ? parsed : 1]
  }))
}

function sameFilters<TFilters extends Record<string, unknown>>(
  left: TFilters,
  right: TFilters,
): boolean {
  return JSON.stringify(left) === JSON.stringify(right)
}

function samePages(left: AdminPageState, right: AdminPageState): boolean {
  return Object.keys(left).length === Object.keys(right).length &&
    Object.keys(left).every(key => left[key] === right[key])
}

/**
 * The URL owns applied filters only. Draft edits stay local until submit, so a
 * shared link always restores the query whose results are currently visible.
 */
export function useAdminQueryState<TFilters extends Record<string, unknown>>(
  options: AdminQueryStateOptions<TFilters>,
) {
  const route = useRoute()
  const router = useRouter()

  const applied = ref<TFilters>(options.decode(route.query))
  const draft = ref<TFilters>(cloneFilters(applied.value))
  const pages = ref<AdminPageState>(decodePageState(route.query, options.pageKeys))


  async function applyRouteState(source: 'initial' | 'route', force = false) {
    const nextFilters = options.decode(route.query)
    const nextPages = decodePageState(route.query, options.pageKeys)
    if (!force && sameFilters(nextFilters, applied.value) && samePages(nextPages, pages.value)) return

    applied.value = nextFilters
    draft.value = cloneFilters(nextFilters)
    pages.value = nextPages
    await options.onApplied(nextFilters, nextPages, source)
  }

  watch(
    () => route.fullPath,
    () => { void applyRouteState('route') },
  )

  async function initialize() {
    await applyRouteState('initial', true)
  }

  async function submit(nextPages: AdminPageState = {}) {
    applied.value = cloneFilters(draft.value)
    pages.value = { ...pages.value, ...nextPages }
    await router.replace(canonicalQuery(options.encode(applied.value, pages.value), {
      ...Object.fromEntries(options.pageKeys.map(key => [key, 1])),
      ...options.defaults,
    }))
    await options.onApplied(applied.value, pages.value, 'submit')
  }

  async function setPage(pageKey: string, page: number) {
    const nextPage = Math.max(1, page)
    if (pages.value[pageKey] === nextPage) return
    pages.value = { ...pages.value, [pageKey]: nextPage }
    await router.replace(canonicalQuery(options.encode(applied.value, pages.value), {
      ...Object.fromEntries(options.pageKeys.map(key => [key, 1])),
      ...options.defaults,
    }))
    await options.onApplied(applied.value, pages.value, 'submit')
  }

  return { draft, applied, pages, initialize, submit, setPage }
}
