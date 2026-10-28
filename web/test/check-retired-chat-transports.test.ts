import { describe, expect, it } from 'vitest'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { findRetiredChatTransports } from '../scripts/check-retired-chat-transports.mjs'

describe('check-retired-chat-transports', () => {
  it('passes when product code uses only the BFF transport', () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'chat-transport-clean-'))
    fs.mkdirSync(path.join(root, 'web/app/src'), { recursive: true })
    fs.writeFileSync(path.join(root, 'web/app/src/example.ts'), "fetch('/api/chat/sessions/1/ag-ui')\n")
    expect(findRetiredChatTransports(root)).toEqual([])
  })

  it('rejects retired direct runtime routes and build switches', () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'chat-transport-retired-'))
    fs.mkdirSync(path.join(root, 'web/app/src'), { recursive: true })
    fs.writeFileSync(path.join(root, 'web/app/src/example.ts'), "fetch('/rt/natal/ag-ui')\n")
    fs.writeFileSync(path.join(root, 'web/app/src/build.ts'), "const transport = 'VITE_CHAT_TRANSPORT'\n")
    const findings = findRetiredChatTransports(root)
    expect(findings).toHaveLength(2)
    expect(findings).toEqual([
      expect.stringContaining('web/app/src/build.ts:1'),
      expect.stringContaining('web/app/src/example.ts:1'),
    ])
    expect(findings.join('\n')).toContain('VITE_CHAT_TRANSPORT')
    expect(findings.join('\n')).toContain('"/rt/"')
  })
})
