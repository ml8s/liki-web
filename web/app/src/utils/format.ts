// 面向用户的展示格式化：避免原始 ID / ISO 时间戳泄漏到界面。
import { i18n } from '@/i18n'

// locale 可选：admin 运营台需要固定输出语言时传入 'zh-Hans'，默认仍跟随客户 locale。
export function formatDateTime(value?: string | null, locale?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(locale ?? i18n.global.locale.value, { hour12: false })
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
