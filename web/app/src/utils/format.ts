// 面向用户的展示格式化：避免原始 ID / ISO 时间戳泄漏到界面。

export function formatDateTime(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

export function productName(productId: string): string {
  const names: Record<string, string> = { 'liki-chat': 'Liki Chat' }
  return names[productId] ?? productId
}