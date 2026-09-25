// 面向用户的展示格式化：避免原始 ID / ISO 时间戳泄漏到界面。
import { i18n } from '@/i18n'

export function formatDateTime(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(i18n.global.locale.value, { hour12: false })
}

export function formatMinorUnit(amount: number, currency: string): string {
  return new Intl.NumberFormat(i18n.global.locale.value, {
    style: 'currency',
    currency,
  }).format(amount / 100)
}

export function productName(productId: string): string {
  const names: Record<string, string> = { 'liki-chat': 'Liki Chat' }
  return names[productId] ?? productId
}

/**
 * 从路由 query 中提取第一个字符串值。
 * query 可能是 string | string[] | undefined，统一返回 string。
 */
export function toFirstQueryString(value: unknown): string {
  if (Array.isArray(value)) return String(value[0] ?? '')
  return String(value ?? '')
}
