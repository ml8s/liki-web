import { describe, expect, it } from 'vitest'
import en from '@/locales/en.json'
import zhHans from '@/locales/zh-Hans.json'
import zhHant from '@/locales/zh-Hant.json'
import { setAppLocale } from '@/i18n'
import { ApiError } from '@/api/http'
import { toUserMessage } from '@/utils/errors'

function flatten(value: unknown, prefix = ''): string[] {
  if (typeof value !== 'object' || value === null) return [prefix]
  return Object.entries(value).flatMap(([key, item]) => flatten(item, prefix ? `${prefix}.${key}` : key))
}

describe('app i18n', () => {
  it('keeps locale keys aligned', () => {
    const locales = { zhHans, zhHant, en }
    const keys = Object.fromEntries(Object.entries(locales).map(([name, data]) => [name, flatten(data)]))
    const expected = new Set(keys.zhHans)

    for (const [name, values] of Object.entries(keys)) {
      for (const key of values) expect(expected.has(key), `${name}:${key}`).toBe(true)
    }
    for (const key of expected) {
      expect(keys.zhHant.includes(key), `zh-Hant:${key}`).toBe(true)
      expect(keys.en.includes(key), `en:${key}`).toBe(true)
    }
  })

  it('keeps critical error codes localized in every locale', () => {
    const criticalCodes = [
      'network_error',
      'invalid_response',
      'request_failed',
      'unauthorized',
      'forbidden',
      'not_found',
      'validation_error',
      'rate_limited',
      'internal_error',
      'checkout_failed',
      'order_failed',
      'generation_failed',
      'generation_cancelled',
      'engine_unavailable',
      'engine_rpc_failed',
    ]

    const localizedErrors: Record<string, Record<string, string>> = {
      'zh-Hans': zhHans.errors.code,
      'zh-Hant': zhHant.errors.code,
      en: en.errors.code,
    }
    for (const [locale, messages] of Object.entries(localizedErrors)) {
      for (const code of criticalCodes) {
        expect(messages[code], `${locale}:${code}`).toBeTruthy()
      }
    }
  })

  it('localizes stable error codes instead of echoing diagnostic messages', async () => {
    await setAppLocale('en', { persist: false })
    try {
      const error = new ApiError(500, 'engine_unavailable', '服务端诊断信息')
      expect(toUserMessage(error, 'Fallback')).toBe(en.errors.code.engine_unavailable)
    } finally {
      await setAppLocale('zh-Hans', { persist: false })
    }
  })
})
