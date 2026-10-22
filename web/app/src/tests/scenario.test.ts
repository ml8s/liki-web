import { describe, expect, it } from 'vitest'
import { normalizeScenario, scenarioService } from '@/composables/scenario'

describe('scenario handoff', () => {
  it('normalizes public marketing scenarios', () => {
    expect(normalizeScenario('natal')).toBe('natal')
    expect(normalizeScenario('liunian')).toBe('liunian')
    expect(normalizeScenario('hepan')).toBe('hepan')
    // 旧参数兼容：bazi/mingshu 并入 natal。
    expect(normalizeScenario('bazi')).toBe('natal')
    expect(normalizeScenario('mingshu')).toBe('natal')
    expect(normalizeScenario('unknown')).toBe('general')
  })

  it('maps scenarios to entry keys', () => {
    expect(scenarioService('naming')).toBe('naming')
    expect(scenarioService('hepan')).toBe('hepan')
    expect(scenarioService('natal')).toBe('natal')
    expect(scenarioService('general')).toBeNull()
  })
})
