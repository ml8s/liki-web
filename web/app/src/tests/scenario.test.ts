import { describe, expect, it } from 'vitest'
import { normalizeScenario, scenarioService } from '@/composables/scenario'

describe('scenario handoff', () => {
  it('normalizes public marketing scenarios', () => {
    expect(normalizeScenario('natal')).toBe('natal')
    expect(normalizeScenario('liunian')).toBe('liunian')
    expect(normalizeScenario('hepan')).toBe('hepan')
    expect(normalizeScenario('bazi')).toBe('general')
    expect(normalizeScenario('mingshu')).toBe('general')
    expect(normalizeScenario('unknown')).toBe('general')
  })

  it('maps scenarios to entry keys', () => {
    expect(scenarioService('naming')).toBe('naming')
    expect(scenarioService('hepan')).toBe('hepan')
    expect(scenarioService('natal')).toBe('natal')
    expect(scenarioService('general')).toBeNull()
  })
})
