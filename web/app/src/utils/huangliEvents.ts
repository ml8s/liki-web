import zhHansMessages from '@/locales/zh-Hans.json'

/**
 * The engine API currently expects the canonical Simplified-Chinese event
 * label. Keep that wire contract independent of the user's interface locale.
 */
export function huangliApiEventLabel(key: string): string {
  const parts = `services.huangli.events.${key}`.split('.')
  let value: unknown = zhHansMessages
  for (const part of parts) {
    value = typeof value === 'object' && value !== null
      ? (value as Record<string, unknown>)[part]
      : undefined
    if (value === undefined) return key
  }
  return typeof value === 'string' ? value : key
}
