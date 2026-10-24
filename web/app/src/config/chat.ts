import { i18n } from '@/i18n'
import scenarioContract from '@shared/config/scenarios.json'

/** 内部专家部署（四专家独立部署；会话创建时按场景路由，不呈现给用户）。 */
export type ExpertId = (typeof scenarioContract.experts)[number]

/** 用户面价值场景（七场景）：与营销页 slug 一一对应（一页一入口）。 */
export type ScenarioId = (typeof scenarioContract.scenarios)[number]['id']

export interface ChatScenario {
  id: ScenarioId
  /** 入口图标字（命/运/合/卦/吉/名/宅；en 为首字母）。 */
  glyphKey: string
  labelKey: string
  descKey: string
  /** 会话首条意图消息（发给场景对应专家的开场陈述）。 */
  intentKey: string
  /** 免费工具页路由（如命理 → 人列表排盘）；无路由的入口走对话。 */
  route?: string
  /** 卡片主按钮文案，避免所有场景都显示“开始”。 */
  ctaKey: string
  /** 会话绑定的内部专家。 */
  expert: ExpertId
}

// 价值入口：按用户场景命名（命理/流年/合盘/问卦/择日/风水/起名），不按引擎 skill。
// route 来自 scenario contract 的 public App path；UI key 由稳定 scenario id 生成。
export const chatScenarios: ChatScenario[] = scenarioContract.scenarios.map(scenario => ({
  id: scenario.id as ScenarioId,
  glyphKey: `services.glyph.${scenario.id}`,
  labelKey: `services.${scenario.id}.label`,
  descKey: `services.${scenario.id}.description`,
  intentKey: `services.${scenario.id}.intent`,
  route: scenario.appPath ? scenario.appPath.replace(/^\/app/, '') || '/' : undefined,
  ctaKey: `services.${scenario.id}.cta`,
  expert: scenario.expert as ExpertId,
}))

export function scenarioById(key: string | null | undefined): ChatScenario | undefined {
  if (!key) return undefined
  return chatScenarios.find(s => s.id === key)
}

export function scenarioByIntent(intent: string): ChatScenario | undefined {
  return chatScenarios.find(s => i18n.global.t(s.intentKey) === intent)
}

// 入口行数据（图标字/名称/说明/动作）：三处视图共用的展示投影。
export function scenarioEntryItems(t: (key: string) => string) {
  return chatScenarios.map(s => ({
    id: s.id,
    glyph: t(s.glyphKey),
    label: t(s.labelKey),
    desc: t(s.descKey),
    action: t(s.ctaKey),
  }))
}

export function chatUrl(profileId: string, scenario: string): string {
  return `/chat?profile=${encodeURIComponent(profileId)}&service=${encodeURIComponent(scenario)}`
}
