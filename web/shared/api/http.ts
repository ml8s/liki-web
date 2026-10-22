// 共享 HTTP 错误类：营销站（Astro island）与 App SPA 共用同一错误契约。
// App 的 api/http.ts 从这里 re-export，其余请求封装（登录跳转等）留在 App 侧。

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
