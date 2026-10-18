import { i18n } from '@/i18n'

export type ExpertId = 'natal' | 'divination' | 'fengshui' | 'naming'

export interface SkillDefinition {
  id: ExpertId
  labelKey: string
  descriptionKey: string
}

// 四专家独立部署：会话在创建时绑定专家，BFF 按专家路由到对应部署
// （endpoint 即专家身份，不再有 LLM 路由层）。
export const chatSkills: SkillDefinition[] = [
  {
    id: 'natal',
    labelKey: 'services.expert.natal.label',
    descriptionKey: 'services.expert.natal.description',
  },
  {
    id: 'divination',
    labelKey: 'services.expert.divination.label',
    descriptionKey: 'services.expert.divination.description',
  },
  {
    id: 'fengshui',
    labelKey: 'services.expert.fengshui.label',
    descriptionKey: 'services.expert.fengshui.description',
  },
  {
    id: 'naming',
    labelKey: 'services.expert.naming.label',
    descriptionKey: 'services.expert.naming.description',
  },
]

export interface ChatService {
  key: string
  labelKey: string
  descKey: string
  /** 会话首条意图消息（发给所选专家的开场陈述） */
  intentKey: string
  /** 特殊入口路由（如合盘 → /synastry），否则走对话意图 */
  route?: string
  /** 卡片主按钮文案，避免所有场景都显示“开始” */
  ctaKey: string
}

// 价值入口：按用户场景命名（看命/合盘/起名/问卦/风水），不按引擎 skill。
// route：每个入口都有免费盘面页（看命/风水→人列表排盘、起名/问卦/合盘→独立预览页）；付费解读统一进 Chat。
export const chatServices: ChatService[] = [
  { key: 'bazi', labelKey: 'services.bazi.label', descKey: 'services.bazi.description', intentKey: 'services.bazi.intent', route: '/birth-profiles', ctaKey: 'services.bazi.cta' },
  { key: 'divination', labelKey: 'services.divination.label', descKey: 'services.divination.description', intentKey: 'services.divination.intent', route: '/divination', ctaKey: 'services.divination.cta' },
  { key: 'fengshui', labelKey: 'services.fengshui.label', descKey: 'services.fengshui.description', intentKey: 'services.fengshui.intent', route: '/fengshui', ctaKey: 'services.fengshui.cta' },
  { key: 'huangli', labelKey: 'services.huangli.label', descKey: 'services.huangli.description', intentKey: 'services.huangli.intent', route: '/huangli', ctaKey: 'services.huangli.cta' },
  { key: 'bond', labelKey: 'services.bond.label', descKey: 'services.bond.description', intentKey: 'services.bond.intent', route: '/synastry', ctaKey: 'services.bond.cta' },
]

export function serviceByKey(key: string): ChatService | undefined {
  return chatServices.find(s => s.key === key)
}

export function serviceByIntent(intent: string): ChatService | undefined {
  return chatServices.find(s => i18n.global.t(s.intentKey) === intent)
}

export function chatUrl(profileId: string, service: string): string {
  return `/chat?profile=${encodeURIComponent(profileId)}&service=${encodeURIComponent(service)}`
}
