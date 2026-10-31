import type { IntentId as ValueIntentId } from '@/config/chat'

export type IntentId = 'general' | ValueIntentId

/** 营销落地页交接参数 → 场景 id；只接受现行七场景。 */
export function normalizeIntent(value: unknown): IntentId {
  const intent = String(value ?? '')
  if (
    intent === 'natal' || intent === 'liunian' || intent === 'hepan' ||
    intent === 'divination' || intent === 'huangli' || intent === 'fengshui' ||
    intent === 'naming'
  ) {
    return intent
  }
  return 'general'
}

// 落地页场景 → 入口 id：场景即入口（营销页与 App 入口 1:1）。
// 有免费工具页的入口由调用方按 chatIntents.route 跳转，无路由的进对话。
export function intentService(intent: IntentId): string | null {
  if (intent === 'general') return null
  return intent
}
