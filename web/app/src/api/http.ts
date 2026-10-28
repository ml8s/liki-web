// ApiError 已迁至共享层（营销站与 App 共用同一错误契约），此处 re-export。
export { ApiError } from '@shared/api/http'
import { ApiError } from '@shared/api/http'

interface Envelope<T> {
  data?: T
}

/** RFC 9457 problem+json：type 为机器契约码，detail 为人读说明。 */
interface ProblemDetails {
  type?: string
  title?: string
  status?: number
  detail?: string
}

// 会话过期（401）→ 回登录页；/api/me 交给路由守卫软跳转，避免整页刷新。
export function redirectToLogin() {
  const base = import.meta.env.BASE_URL || '/app/'
  const path = location.pathname.startsWith(base)
    ? `/${location.pathname.slice(base.length)}${location.search}`
    : location.pathname + location.search
  window.location.assign(`${base}login?redirect=${encodeURIComponent(path)}`)
}

export type HttpRequestInit = RequestInit & {
  timeoutMs?: number
  retries?: number
}

export async function request<T>(path: string, init: HttpRequestInit = {}): Promise<T> {
  const { timeoutMs = 12_000, retries, ...fetchInit } = init
  const retryLimit = retries ?? ((fetchInit.method ?? 'GET') === 'GET' ? 2 : 0)
  let lastError: unknown

  for (let attempt = 0; attempt <= retryLimit; attempt += 1) {
    try {
      return await requestOnce<T>(path, fetchInit, timeoutMs)
    } catch (error) {
      lastError = error
      const retryable = error instanceof ApiError
        && [0, 502, 503, 504].includes(error.status)
      if (!retryable || attempt === retryLimit) break
      await new Promise(resolve => setTimeout(resolve, 150 * 2 ** attempt))
    }
  }

  throw lastError
}

async function requestOnce<T>(
  path: string,
  init: RequestInit,
  timeoutMs: number,
): Promise<T> {
  const timeoutController = new AbortController()
  const timeoutId = setTimeout(() => {
    timeoutController.abort(new DOMException('Request timed out', 'TimeoutError'))
  }, timeoutMs)
  const abortExternal = () => timeoutController.abort(init.signal?.reason)
  init.signal?.addEventListener('abort', abortExternal, { once: true })

  let response: Response
  try {
    response = await fetch(path, {
      credentials: 'same-origin',
      headers: init.body ? { 'Content-Type': 'application/json' } : undefined,
      ...init,
      signal: timeoutController.signal,
    })
  } catch {
    if (init.signal?.aborted) {
      throw init.signal.reason ?? new DOMException('Request aborted', 'AbortError')
    }
    const timedOut = timeoutController.signal.reason instanceof DOMException
      && timeoutController.signal.reason.name === 'TimeoutError'
    // 网络异常只保留稳定 code；用户提示由 errors.code.network_error 本地化。
    throw new ApiError(0, 'network_error', timedOut
      ? 'Request timed out'
      : 'Network request failed')
  } finally {
    clearTimeout(timeoutId)
    init.signal?.removeEventListener('abort', abortExternal)
  }

  let envelope: Envelope<T> | null
  try {
    envelope = await response.json()
  } catch {
    envelope = null
  }

  if (!response.ok) {
    if (response.status === 401 && path !== '/api/me') {
      redirectToLogin()
    }
    let problem: ProblemDetails | null = null
    if (envelope && typeof envelope === 'object' && 'type' in envelope) {
      problem = envelope as unknown as ProblemDetails
    }
    throw new ApiError(
      response.status,
      problem?.type ?? 'request_failed',
      problem?.detail ?? problem?.title ?? `HTTP ${response.status}`,
    )
  }
  if (!envelope || !('data' in envelope)) {
    throw new ApiError(response.status, 'invalid_response', 'Invalid API response')
  }
  return envelope.data as T
}

export function get<T>(path: string): Promise<T> {
  return request<T>(path)
}

export function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    body: JSON.stringify(body ?? {}),
  })
}

export function patch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'PATCH',
    body: JSON.stringify(body ?? {}),
  })
}

export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}
