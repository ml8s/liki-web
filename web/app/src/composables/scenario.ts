export type ScenarioId = 'general' | 'naming' | 'mingshu' | 'hepan'

export function normalizeScenario(value: unknown): ScenarioId {
  const scenario = String(value ?? '')
  return scenario === 'naming' || scenario === 'mingshu' || scenario === 'hepan' ? scenario : 'general'
}

// 落地页场景 → chat 服务映射：hepan 复用合盘(bond)，mingshu 复用看命(bazi)，
// naming 直连起名入口（四专家之一）。
export function scenarioService(scenario: ScenarioId): string | null {
  if (scenario === 'hepan') return 'bond'
  if (scenario === 'mingshu') return 'bazi'
  if (scenario === 'naming') return 'naming'
  return null
}