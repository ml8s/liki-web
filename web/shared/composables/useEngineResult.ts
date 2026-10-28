import { computed, getCurrentScope, onScopeDispose, ref, watch, type Ref } from 'vue'

export type EngineResultStatus = 'idle' | 'loading' | 'success' | 'error'

export interface EngineResultFailure {
  name: 'EngineResultError'
  message: string
  /** Normalized API error code, so the UI layer can translate it. */
  code?: string
  cause?: unknown
}

export interface EngineResultRecord {
  data: unknown
  fetchedAt: number
}

export type EngineResultSource<T> = T | { readonly value: T } | (() => T)

interface UseEngineResultOptions<TInput, TResult> {
  input: EngineResultSource<TInput | null | undefined>
  load: (input: TInput) => Promise<TResult>
  /**
   * Query namespace, normally the resource id. Prefixed to the canonical input
   * hash so two methods that accept the same DTO cannot share a cache entry.
   */
  namespace?: EngineResultSource<string | null | undefined>
  cacheKey?: EngineResultSource<string | null | undefined>
  enabled?: EngineResultSource<boolean>
  staleTime?: number
  immediate?: boolean
  onError?: (failure: EngineResultFailure) => void
}

export interface UseEngineResultResource<TResult> {
  status: Readonly<Ref<EngineResultStatus>>
  data: Readonly<Ref<TResult | null>>
  error: Readonly<Ref<EngineResultFailure | null>>
  refreshing: Readonly<Ref<boolean>>
  lastUpdatedAt: Readonly<Ref<number | null>>
  cacheKey: Readonly<Ref<string | null>>
  reload: () => Promise<TResult | null>
  refresh: () => Promise<TResult | null>
  invalidate: () => void
  /** Drops the active result and returns to idle without touching the cache. */
  reset: () => void
}

interface EngineCacheRecord {
  data: unknown
  fetchedAt: number
}

const cacheRecords = new Map<string, EngineCacheRecord>()
const cachePromises = new Map<string, Promise<unknown>>()
const DEFAULT_STALE_TIME = 5 * 60_000

function stableStringify(value: unknown): string {
  if (value === undefined) return 'undefined'
  if (value === null) return 'null'
  if (typeof value === 'function' || typeof value === 'symbol') {
    throw new TypeError('Engine result input contains a non-serializable value')
  }
  if (value instanceof Date) return JSON.stringify(value.toISOString())
  if (typeof value === 'bigint') return JSON.stringify(value.toString())
  if (typeof value === 'number') {
    return Number.isFinite(value) ? JSON.stringify(value) : JSON.stringify(null)
  }
  if (typeof value === 'boolean' || typeof value === 'string') {
    return JSON.stringify(value)
  }
  if (Array.isArray(value)) {
    return `[${value.map(item => stableStringify(item)).join(',')}]`
  }
  if (typeof value !== 'object') {
    throw new TypeError(`Engine result input has an unsupported type: ${typeof value}`)
  }
  const prototype = Object.getPrototypeOf(value)
  if (prototype !== Object.prototype && prototype !== null) {
    throw new TypeError('Engine result input must use plain objects')
  }
  const entries = Object.entries(value)
    .filter(([, item]) => item !== undefined)
    .sort(([left], [right]) => left.localeCompare(right))
  return `{${entries.map(([key, item]) => `${JSON.stringify(key)}:${stableStringify(item)}`).join(',')}}`
}

function readCache<TResult>(key: string) {
  return cacheRecords.get(key) as EngineCacheRecord & { data: TResult } | undefined
}

function writeCache(key: string, data: unknown): void {
  // Engine inputs and charts may contain personal birth/profile facts. This
  // cache is intentionally in-memory only and must never move to storage.
  cacheRecords.set(key, { data, fetchedAt: Date.now() })
}

function deleteCache(key: string): void {
  cacheRecords.delete(key)
  cachePromises.delete(key)
}

export function clearEngineResultCache(): void {
  cacheRecords.clear()
  cachePromises.clear()
}

function failureCode(error: unknown): string | undefined {
  if (error === null || typeof error !== 'object') return undefined
  if ('code' in error) {
    const code = (error as { code?: unknown }).code
    if (typeof code === 'string' && code) return code
  }
  // Domain stage wrappers (for example NatalStageError) keep the transport
  // error in `cause`; surface its normalized code for UI translation.
  if ('cause' in error) return failureCode((error as { cause?: unknown }).cause)
  return undefined
}

function toFailure(error: unknown): EngineResultFailure {
  const code = failureCode(error)
  if (error instanceof Error) {
    return {
      name: 'EngineResultError',
      message: error.message,
      code,
      cause: error.cause,
    }
  }
  return {
    name: 'EngineResultError',
    message: String(error),
    code,
    cause: error,
  }
}

function inputHash(value: unknown): string | null {
  try {
    return stableStringify(value)
  } catch {
    return null
  }
}

export function useEngineResult<TInput, TResult>(
  options: UseEngineResultOptions<TInput, TResult>,
): UseEngineResultResource<TResult> {
  const data = ref<TResult | null>(null) as Ref<TResult | null>
  const error = ref<EngineResultFailure | null>(null)
  const status = ref<EngineResultStatus>('idle')
  const refreshing = ref(false)
  const lastUpdatedAt = ref<number | null>(null)
  const activeKey = ref<string | null>(null)
  const activeInputHash = ref<string | null>(null)
  let requestToken = 0

  function staleTime(): number {
    return Math.max(0, options.staleTime ?? DEFAULT_STALE_TIME)
  }

  function readSource(source: EngineResultSource<unknown>): unknown {
  if (typeof source === 'function') return source()
  if (source !== null && typeof source === 'object' && 'value' in source) return source.value
  return source
}

  function currentInputHash(): string | null {
    return inputHash(readSource(options.input))
  }

  function currentCacheKey(): string | null {
    const explicitKey = readSource(options.cacheKey)
    const baseKey = typeof explicitKey === 'string' ? explicitKey : currentInputHash()
    if (!baseKey) return null
    const namespace = readSource(options.namespace)
    return typeof namespace === 'string' && namespace ? `${namespace}:${baseKey}` : baseKey
  }

  const cacheKey = computed(currentCacheKey)

  function applyCache(record: EngineCacheRecord & { data: TResult }): void {
    data.value = record.data
    error.value = null
    refreshing.value = false
    status.value = 'success'
    lastUpdatedAt.value = record.fetchedAt
  }

  async function execute(mode: 'refresh' | 'reload' = 'refresh'): Promise<TResult | null> {
    const enabled = readSource(options.enabled) !== false
    const input = readSource(options.input)
    const key = currentCacheKey()
    const inputKey = currentInputHash()

    if (!enabled || input === null || input === undefined || !key || !inputKey) {
      status.value = 'idle'
      return null
    }

    const token = ++requestToken
    activeKey.value = key
    activeInputHash.value = inputKey

    const cached = readCache<TResult>(key)
    const age = cached ? Date.now() - cached.fetchedAt : Number.POSITIVE_INFINITY
    const isFresh = mode === 'refresh' && age < staleTime()

    if (isFresh) {
      applyCache(cached!)
      return cached!.data
    }

    if (cached) {
      data.value = cached.data
      lastUpdatedAt.value = cached.fetchedAt
      refreshing.value = true
    } else {
      data.value = null
      error.value = null
      refreshing.value = true
      status.value = 'loading'
    }

    const existingPromise = cachePromises.get(key)
    const requestPromise = (existingPromise ?? Promise.resolve().then(() =>
      options.load(input as TInput),
    )).then((result) => {
      if (cachePromises.get(key) === requestPromise) cachePromises.delete(key)
      writeCache(key, result)
      return result as TResult
    }, (cause) => {
      if (cachePromises.get(key) === requestPromise) cachePromises.delete(key)
      throw cause
    })

    if (!existingPromise) cachePromises.set(key, requestPromise)

    try {
      const result = await requestPromise
      if (token !== requestToken) return null
      if (activeKey.value !== key || activeInputHash.value !== inputKey) return null

      data.value = result
      error.value = null
      refreshing.value = false
      lastUpdatedAt.value = Date.now()
      status.value = 'success'
      return result
    } catch (cause) {
      const failure = toFailure(cause)
      if (token !== requestToken) return null
      if (activeKey.value !== key || activeInputHash.value !== inputKey) return null

      error.value = failure
      refreshing.value = false
      status.value = 'error'
      options.onError?.(failure)
      return null
    }
  }

  async function reload(): Promise<TResult | null> {
    return execute('reload')
  }

  async function refresh(): Promise<TResult | null> {
    return execute('refresh')
  }

  function invalidate(): void {
    const key = currentCacheKey()
    if (key) deleteCache(key)
    void execute('reload')
  }

  function reset(): void {
    requestToken += 1
    data.value = null
    error.value = null
    refreshing.value = false
    status.value = 'idle'
    activeKey.value = null
    activeInputHash.value = null
  }

  watch(
    () => readSource(options.input),
    () => {
      void execute('refresh')
    },
    { deep: true },
  )

  watch(
    () => readSource(options.enabled),
    (enabled) => {
      if (enabled) void execute('refresh')
      else status.value = 'idle'
    },
  )

  if (options.immediate !== false) void execute('refresh')

  if (getCurrentScope()) {
    onScopeDispose(() => {
      requestToken += 1
    })
  }

  return {
    status,
    data,
    error,
    refreshing,
    lastUpdatedAt,
    cacheKey,
    reload,
    refresh,
    invalidate,
    reset,
  }
}
