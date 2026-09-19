// 统一的用户可见错误消息层：
//  - 中文消息（后端 ApiError、自定义错误、引擎错误）直接展示
//  - 网络异常（Failed to fetch / NetworkError）→ 友好提示
//  - 其他技术性错误（Unexpected token 等）→ 不回显原始 message，降级回退
const CJK = /[\u4e00-\u9fff]/

export function toUserMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    const message = error.message?.trim()
    if (message) {
      if (CJK.test(message)) return message
      if (error.name === 'TypeError' || error.name === 'NetworkError') {
        return '网络连接失败，请检查网络后重试'
      }
    }
  }
  return fallback
}

/** 统一的 store 错误消息函数，内部委托 toUserMessage。 */
export function errorMessage(error: unknown): string {
  return toUserMessage(error, '请求失败')
}