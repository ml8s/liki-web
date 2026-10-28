import { describe, expect, it } from 'vitest'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { contractErrors, collectGoCodes, renderDoc } from '../scripts/check-error-codes.js'

function fixture({
  localeExtra = [],
  frontendCodes = [],
}: { localeExtra?: string[]; frontendCodes?: string[] } = {}) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'check-error-codes-'))
  fs.mkdirSync(path.join(root, 'internal/example'), { recursive: true })
  fs.mkdirSync(path.join(root, 'web/app/src/locales'), { recursive: true })
  fs.writeFileSync(
    path.join(root, 'internal/example/http.go'),
    'package example\nimport "net/http"\nfunc handler(w http.ResponseWriter) { RespondError(w, http.StatusBadRequest, "kept_code", "invalid") }\n',
  )
  for (const locale of ['zh-Hans', 'zh-Hant', 'en']) {
    const codes = Object.fromEntries(['kept_code', ...localeExtra].map(code => [code, code]))
    fs.writeFileSync(
      path.join(root, `web/app/src/locales/${locale}.json`),
      JSON.stringify({ errors: { code: codes } }),
    )
  }
  if (frontendCodes.length) {
    fs.mkdirSync(path.join(root, 'web/scripts'), { recursive: true })
    fs.writeFileSync(
      path.join(root, 'web/scripts/error-codes-frontend.json'),
      JSON.stringify({ codes: frontendCodes }),
    )
  }
  return root
}

describe('check-error-codes contract', () => {
  it('extracts Go error codes and renders the generated table', () => {
    const root = fixture()
    const codes = collectGoCodes(root)
    expect(codes.get('kept_code')).toBe('internal/example/http.go')
    expect(renderDoc(codes)).toContain('| `kept_code` | `internal/example/http.go` |')
  })

  it('rejects locale-only orphan error codes', () => {
    const root = fixture({ localeExtra: ['retired_code'] })
    expect(contractErrors(root)).toEqual(
      expect.arrayContaining([
        expect.stringContaining('zh-Hans.json: orphan errors.code.retired_code'),
        expect.stringContaining('zh-Hant.json: orphan errors.code.retired_code'),
        expect.stringContaining('en.json: orphan errors.code.retired_code'),
      ]),
    )
  })

  it('keeps locale coverage bidirectional when every code has exactly one source', () => {
    expect(contractErrors(fixture({
      localeExtra: ['frontend_code'],
      frontendCodes: ['frontend_code'],
    }))).toEqual([])
  })

  it('reports frontend-only inventory codes missing from locales', () => {
    expect(contractErrors(fixture({ frontendCodes: ['frontend_code'] }))).toEqual(
      expect.arrayContaining([
        expect.stringContaining('zh-Hans.json: missing errors.code.frontend_code (frontend-only definition)'),
        expect.stringContaining('zh-Hant.json: missing errors.code.frontend_code (frontend-only definition)'),
        expect.stringContaining('en.json: missing errors.code.frontend_code (frontend-only definition)'),
      ]),
    )
  })
})
