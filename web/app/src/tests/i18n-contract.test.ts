import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import en from '@/locales/en.json'
import zhHans from '@/locales/zh-Hans.json'
import zhHant from '@/locales/zh-Hant.json'

const sourceRoot = join(process.cwd(), 'src')
const sourceDirectories = ['api', 'composables', 'config', 'stores', 'utils', 'views', 'components']

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(item => {
    const path = join(directory, item.name)
    if (item.isDirectory()) return sourceFiles(path)
    return /\.(vue|ts)$/.test(item.name) && !item.name.endsWith('.d.ts') ? [path] : []
  })
}

function flatten(value: unknown, prefix = ''): string[] {
  if (typeof value !== 'object' || value === null) return [prefix]
  return Object.entries(value).flatMap(([key, item]) => flatten(item, prefix ? `${prefix}.${key}` : key))
}

describe('i18n source contract', () => {
  it('defines every static translation key in all locales', () => {
    const locales = { 'zh-Hans': zhHans, 'zh-Hant': zhHant, en } as const
    const available = Object.fromEntries(
      Object.entries(locales).map(([name, messages]) => [name, new Set(flatten(messages))]),
    ) as Record<keyof typeof locales, Set<string>>
    const missing: string[] = []

    for (const directory of sourceDirectories) {
      for (const file of sourceFiles(join(sourceRoot, directory))) {
        const source = readFileSync(file, 'utf8')
        for (const match of source.matchAll(/(?:\$t|\bt)\(\s*['"]([^'"\n]+)['"]/g)) {
          const key = match[1]
          if (!key.includes('.')) continue
          for (const [locale, keys] of Object.entries(available)) {
            if (!keys.has(key)) missing.push(`${locale}:${key} (${file})`)
          }
        }
      }
    }

    expect(missing).toEqual([])
  })

  it('defines dynamically constructed partner rule keys in all locales', () => {
    const locales = { 'zh-Hans': zhHans, 'zh-Hant': zhHant, en } as const
    const requiredRoles = ['kol', 'scout'] as const
    const requiredNumbers = [1, 2] as const
    const missing: string[] = []

    for (const [locale, messages] of Object.entries(locales)) {
      const keys = new Set(flatten(messages))
      for (const role of requiredRoles) {
        const cap = role === 'kol' ? 'Kol' : 'Scout'
        for (const number of requiredNumbers) {
          const key = `partner.rule${cap}${number}`
          if (!keys.has(key)) missing.push(`${locale}:${key}`)
        }
      }
    }

    expect(missing).toEqual([])
  })
})
