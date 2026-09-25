import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const pages = ['naming', 'mingshu', 'hepan']
const root = process.cwd()

describe('scenario landing pages', () => {
  it.each(pages)('%s routes users into the authenticated Liki Chat app', name => {
    const source = readFileSync(join(root, `src/pages/[locale]/${name}.astro`), 'utf8')
    expect(source).toContain('ScenarioLanding')
    expect(source).toContain(`scenario="${name}"`)
    expect(source).not.toContain('PaymentRow')
    expect(source).not.toContain('payment.js')
    expect(source).not.toContain('purchase-form')
    expect(source).not.toContain('"@type":"Product"')
  })

  it('builds locale-aware CTAs and service/FAQ structured data', () => {
    const source = readFileSync(join(root, 'src/components/ScenarioLanding.astro'), 'utf8')
    expect(source).toContain("'@type': 'Service'")
    expect(source).toContain("'@type': 'FAQPage'")
    const cta = readFileSync(join(root, 'src/components/LikiChatCta.astro'), 'utf8')
    expect(cta).toContain('`/app?scenario=${encodeURIComponent(scenario)}&locale=${encodeURIComponent(locale)}`')
  })
})
