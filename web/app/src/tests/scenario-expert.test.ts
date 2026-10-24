import { describe, expect, it } from 'vitest'
import { chatScenarios, scenarioById } from '@/config/chat'

describe('scenario to expert routing contract', () => {
  it('maps every user-facing scenario to its expert deployment', () => {
    expect(Object.fromEntries(chatScenarios.map(item => [item.id, item.expert]))).toEqual({
      natal: 'natal',
      liunian: 'natal',
      hepan: 'natal',
      divination: 'divination',
      huangli: 'divination',
      fengshui: 'fengshui',
      naming: 'naming',
    })
  })

  it('rejects unknown scenario ids instead of silently defaulting to an expert', () => {
    expect(scenarioById('unknown')).toBeUndefined()
    expect(scenarioById(null)).toBeUndefined()
  })
})
