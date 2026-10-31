import { describe, expect, it } from 'vitest'
import { normalizeIntent, intentService } from '@/composables/intent'

describe('intent handoff', () => {
  it('normalizes public marketing intents', () => {
    expect(normalizeIntent('natal')).toBe('natal')
    expect(normalizeIntent('liunian')).toBe('liunian')
    expect(normalizeIntent('hepan')).toBe('hepan')
    expect(normalizeIntent('bazi')).toBe('general')
    expect(normalizeIntent('mingshu')).toBe('general')
    expect(normalizeIntent('unknown')).toBe('general')
  })

  it('maps intents to entry keys', () => {
    expect(intentService('naming')).toBe('naming')
    expect(intentService('hepan')).toBe('hepan')
    expect(intentService('natal')).toBe('natal')
    expect(intentService('general')).toBeNull()
  })
})
