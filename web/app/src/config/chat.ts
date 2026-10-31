import { i18n } from '@/i18n'
import intentContract from '@shared/config/intents.json'

/** 用户面价值场景（七场景）：与营销页 slug 一一对应（一页一入口）。 */
export type IntentId = (typeof intentContract.intents)[number]['id']

export interface ChatIntent {
  id: IntentId
  /** 入口图标字（命/运/合/卦/吉/名/宅；en 为首字母）。 */
  glyphKey: string
  labelKey: string
  descKey: string
  /** 会话首条意图消息。 */
  intentKey: string
  /** 免费工具页路由（如命理 → 人列表排盘）；无路由的入口走对话。 */
  route?: string
  /** 卡片主按钮文案，避免所有场景都显示“开始”。 */
  ctaKey: string
}

// 价值入口：按用户场景命名（命理/流年/合盘/问卦/择日/风水/起名），不按引擎 skill。
// route 来自 intent contract 的 public App path；UI key 由稳定 intent id 生成。
export const chatIntents: ChatIntent[] = intentContract.intents.map(intent => ({
  id: intent.id as IntentId,
  glyphKey: `services.glyph.${intent.id}`,
  labelKey: `services.${intent.id}.label`,
  descKey: `services.${intent.id}.description`,
  intentKey: `services.${intent.id}.intent`,
  route: intent.appPath ? intent.appPath.replace(/^\/app/, '') || '/' : undefined,
  ctaKey: `services.${intent.id}.cta`,
}))

export function intentById(key: string | null | undefined): ChatIntent | undefined {
  if (!key) return undefined
  return chatIntents.find(s => s.id === key)
}

export function intentByIntent(intent: string): ChatIntent | undefined {
  return chatIntents.find(s => i18n.global.t(s.intentKey) === intent)
}

// 入口行数据（图标字/名称/说明/动作）：三处视图共用的展示投影。
export function intentEntryItems(t: (key: string) => string) {
  return chatIntents.map(s => ({
    id: s.id,
    glyph: t(s.glyphKey),
    label: t(s.labelKey),
    desc: t(s.descKey),
    action: t(s.ctaKey),
  }))
}

export function chatUrl(profileId: string, intent: string): string {
  return `/chat?profile=${encodeURIComponent(profileId)}&service=${encodeURIComponent(intent)}`
}
