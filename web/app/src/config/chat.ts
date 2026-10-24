import { i18n } from '@/i18n'

/** 内部专家部署（四专家独立部署；会话创建时按场景路由，不呈现给用户）。 */
export type ExpertId = 'natal' | 'divination' | 'fengshui' | 'naming'

/** 用户面价值场景（七场景）：与营销页 slug 一一对应（一页一入口）。 */
export type ScenarioId = 'natal' | 'liunian' | 'hepan' | 'divination' | 'huangli' | 'fengshui' | 'naming'

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
// route：多数入口有免费盘面页（命理/风水→人列表排盘、问卦/择日/合盘→独立工具页）；
// 流年/起名无独立 App 页，付费解读统一进 Chat。
export const chatScenarios: ChatScenario[] = [
  { id: 'natal', glyphKey: 'services.glyph.natal', labelKey: 'services.natal.label', descKey: 'services.natal.description', intentKey: 'services.natal.intent', route: '/birth-profiles', ctaKey: 'services.natal.cta', expert: 'natal' },
  { id: 'liunian', glyphKey: 'services.glyph.liunian', labelKey: 'services.liunian.label', descKey: 'services.liunian.description', intentKey: 'services.liunian.intent', ctaKey: 'services.liunian.cta', expert: 'natal' },
  { id: 'hepan', glyphKey: 'services.glyph.hepan', labelKey: 'services.hepan.label', descKey: 'services.hepan.description', intentKey: 'services.hepan.intent', route: '/hepan', ctaKey: 'services.hepan.cta', expert: 'natal' },
  { id: 'divination', glyphKey: 'services.glyph.divination', labelKey: 'services.divination.label', descKey: 'services.divination.description', intentKey: 'services.divination.intent', route: '/divination', ctaKey: 'services.divination.cta', expert: 'divination' },
  { id: 'huangli', glyphKey: 'services.glyph.huangli', labelKey: 'services.huangli.label', descKey: 'services.huangli.description', intentKey: 'services.huangli.intent', route: '/huangli', ctaKey: 'services.huangli.cta', expert: 'divination' },
  { id: 'fengshui', glyphKey: 'services.glyph.fengshui', labelKey: 'services.fengshui.label', descKey: 'services.fengshui.description', intentKey: 'services.fengshui.intent', route: '/fengshui', ctaKey: 'services.fengshui.cta', expert: 'fengshui' },
  { id: 'naming', glyphKey: 'services.glyph.naming', labelKey: 'services.naming.label', descKey: 'services.naming.description', intentKey: 'services.naming.intent', ctaKey: 'services.naming.cta', expert: 'naming' },
]

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
