export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

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

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, {
      credentials: 'same-origin',
      headers: init.body ? { 'Content-Type': 'application/json' } : undefined,
      ...init,
    })
  } catch {
    // 网络异常只保留稳定 code；用户提示由 errors.code.network_error 本地化。
    throw new ApiError(0, 'network_error', 'Network request failed')
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

export function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}
