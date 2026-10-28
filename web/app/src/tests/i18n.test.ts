import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
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

// Every key the product code references, as literals plus the template families it
// builds at runtime (t(`errors.code.${code}`)). Deliberately derived from source
// rather than maintained by hand: a key counts as used because the source says so.
function referencedKeys(): (key: string) => boolean {
  const literals = new Set<string>()
  const prefixes = new Set<string>()
  const walk = (dir: string) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      // The catalogs themselves and the tests are not product consumers.
      if (entry.isDirectory()) {
        if (entry.name !== 'locales' && entry.name !== 'tests') walk(join(dir, entry.name))
        continue
      }
      if (!/\.[cm]?[jt]s$|\.vue$/.test(entry.name)) continue
      const source = readFileSync(join(dir, entry.name), 'utf-8')
      for (const re of [/'([^'\n]*)'/g, /"([^"\n]*)"/g]) {
        for (const m of source.matchAll(re)) literals.add(m[1])
      }
      for (const m of source.matchAll(/`([^`\n]*)`/g)) {
        const prefix = m[1].split('${')[0]
        if (m[1].includes('${') && prefix) prefixes.add(prefix)
      }
    }
  }
  // vitest 从 web/app 运行（`npm run test` / Makefile 都先 cd web/app），
  // 所以 cwd 就是应用根；路径不存在会直接 ENOENT 报错，不会静默跳过。
  walk(join(process.cwd(), 'src'))
  // 排盘共享组件已迁至 web/shared（营销站 island 与 App 共用），
  // 其引用的键同样由 App 的 locale 目录供给，纳入扫描。
  walk(join(process.cwd(), '../shared'))
  return (key) => literals.has(key) || [...prefixes].some((prefix) => key.startsWith(prefix))
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

  it('references every key from product code', () => {
    const isReferenced = referencedKeys()
    const unused = flatten(zhHans).filter((key) => !isReferenced(key))
    expect(unused, `unused keys: ${unused.join(', ')}`).toEqual([])
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
      'order_not_pending',
      'order_expired',
      'order_already_paid',
      'order_refunded',
      'generation_failed',
      'generation_cancelled',
      'engine_unavailable',
      'engine_tool_failed',
      // BFF AG-UI 路径可达的 client/runtime/协议错误码。前端本地生成或
      // 经 RUN_ERROR.code 透传；完整清单见 error-codes-frontend.json。
      'runtime_init_failed',
      'runtime_session_unavailable',
      'runtime_busy',
      'runtime_empty_response',
      'runtime_failed',
      'runtime_interrupted',
      'runtime_timeout',
      'llm_unavailable',
      'llm_request_invalid',
      'llm_call_failed',
      'mcp_tools_unavailable',
      'tool_call_invalid',
      'tool_call_unknown',
      'tool_execution_failed',
      'tool_execution_interrupted',
      'agent_definition_invalid',
      'agent_definition_missing',
      'agent_delegation_unknown',
      'agent_delegation_failed',
      'structured_output_invalid',
      'structured_output_empty',
      'user_message_required',
      'history_invalid',
      'run_id_conflict',
      'run_id_required',
      'run_id_invalid',
      'thread_id_required',
      'thread_id_invalid',
      'user_id_required',
      'user_id_invalid',
      'identity_required',
      'method_not_allowed',
      'invalid_agui_request',
      'unsupported_agui_feature',
      'invalid_agui_message',
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
