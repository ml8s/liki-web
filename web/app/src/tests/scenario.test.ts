import { describe, expect, it } from 'vitest'
import { normalizeScenario, scenarioService } from '@/composables/scenario'

describe('scenario handoff', () => {
  it('normalizes public marketing scenarios', () => {
    expect(normalizeScenario('naming')).toBe('naming')
    expect(normalizeScenario('mingshu')).toBe('mingshu')
    expect(normalizeScenario('hepan')).toBe('hepan')
    expect(normalizeScenario('unknown')).toBe('general')
  })

  it('maps marketing scenarios to chat services', () => {
    expect(scenarioService('naming')).toBeNull()
    expect(scenarioService('hepan')).toBe('bond')
    expect(scenarioService('mingshu')).toBe('bazi')
    expect(scenarioService('general')).toBeNull()
  })
})