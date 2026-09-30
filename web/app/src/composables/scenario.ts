export type ScenarioId = 'general' | 'naming' | 'mingshu' | 'hepan'

export function normalizeScenario(value: unknown): ScenarioId {
  const scenario = String(value ?? '')
  return scenario === 'naming' || scenario === 'mingshu' || scenario === 'hepan' ? scenario : 'general'
}

// 落地页场景 → chat 服务映射：hepan 复用合盘(bond)，mingshu 复用看命(bazi)。
// naming 无对应服务，走默认 Chat 入口。
export function scenarioService(scenario: ScenarioId): string | null {
  if (scenario === 'hepan') return 'bond'
  if (scenario === 'mingshu') return 'bazi'
  return null
}