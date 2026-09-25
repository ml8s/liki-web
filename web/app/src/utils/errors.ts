// 统一的用户可见错误消息层。用户提示只来自 locale 资源；
// 后端/引擎 message 是诊断信息，不直接回显，避免语言漂移。
import { i18n } from '@/i18n'

function t(key: string): string {
  return i18n.global.t(key)
}

export function toUserMessage(error: unknown, fallback: string): string {
  const code = error instanceof Error && 'code' in error ?
    String((error as Error & { code?: unknown }).code ?? '') : ''
  if (code) {
    const key = `errors.code.${code}`
    const translated = i18n.global.te(key) ? i18n.global.t(key) : ''
    if (translated) return translated
  }
  if (error instanceof Error) {
    const message = error.message?.trim()
    if (message && (error.name === 'TypeError' || error.name === 'NetworkError')) {
      return t('errors.network')
    }
  }
  return fallback
}

/** 统一的 store 错误消息函数，内部委托 toUserMessage。 */
export function errorMessage(error: unknown): string {
  return toUserMessage(error, t('errors.requestFailed'))
}
