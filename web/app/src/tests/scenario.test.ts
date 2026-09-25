import { describe, expect, it } from 'vitest'
import { normalizeScenario, resolveScenarioTarget } from '@/composables/scenario'

describe('scenario handoff', () => {
  it('normalizes public marketing scenarios', () => {
    expect(normalizeScenario('naming')).toBe('naming')
    expect(normalizeScenario('mingshu')).toBe('mingshu')
    expect(normalizeScenario('hepan')).toBe('hepan')
    expect(normalizeScenario('unknown')).toBe('general')
  })

  it('resolves profile-aware targets', () => {
    expect(resolveScenarioTarget('naming', [])).toEqual({ path: '/naming' })
    expect(resolveScenarioTarget('hepan', ['p1'])).toEqual({ path: '/synastry' })
    expect(resolveScenarioTarget('mingshu', [])).toEqual({
      path: '/profiles/new', service: 'bazi', storeService: true,
    })
    expect(resolveScenarioTarget('mingshu', ['p1'])).toEqual({
      path: '/chat?profile=p1&service=bazi',
    })
  })
})
