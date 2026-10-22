import type { ScenarioId as ValueScenarioId } from '@/config/chat'

export type ScenarioId = 'general' | ValueScenarioId

/** 营销落地页交接参数 → 场景 id；旧参数（bazi/mingshu）保留兼容。 */
export function normalizeScenario(value: unknown): ScenarioId {
  const scenario = String(value ?? '')
  if (scenario === 'bazi' || scenario === 'mingshu') return 'natal'
  if (
    scenario === 'natal' || scenario === 'liunian' || scenario === 'hepan' ||
    scenario === 'divination' || scenario === 'huangli' || scenario === 'fengshui' ||
    scenario === 'naming'
  ) {
    return scenario
  }
  return 'general'
}

// 落地页场景 → 入口 id：场景即入口（营销页与 App 入口 1:1）。
// 有免费工具页的入口由调用方按 chatScenarios.route 跳转，无路由的进对话。
export function scenarioService(scenario: ScenarioId): string | null {
  if (scenario === 'general') return null
  return scenario
}
