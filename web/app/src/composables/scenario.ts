export type ScenarioId = 'general' | 'naming' | 'mingshu' | 'hepan'

export function normalizeScenario(value: unknown): ScenarioId {
  const scenario = String(value ?? '')
  return scenario === 'naming' || scenario === 'mingshu' || scenario === 'hepan' ? scenario : 'general'
}

export interface ScenarioTarget {
  path: string
  service?: string
  storeService?: boolean
}

export function resolveScenarioTarget(scenario: ScenarioId, profileIds: string[]): ScenarioTarget | null {
  if (scenario === 'naming') return { path: '/naming' }
  if (scenario === 'hepan') return { path: '/synastry' }
  if (scenario === 'mingshu') {
    const profileId = profileIds[0]
    if (!profileId) return { path: '/profiles/new', service: 'bazi', storeService: true }
    return { path: `/chat?profile=${encodeURIComponent(profileId)}&service=bazi` }
  }
  return null
}
