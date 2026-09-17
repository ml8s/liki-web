export interface SkillDefinition {
  id: 'liki'
  label: string
  description: string
}

// 引擎已把四个领域合并为统一 skill「liki」，由引擎按意图自动路由；
// 前端只保留一个入口。
export const chatSkills: SkillDefinition[] = [
  {
    id: 'liki',
    label: '命理咨询',
    description: '八字、紫微、起名、问卦、风水——一个入口，按你的问题自动匹配领域。',
  },
]

export interface ChatService {
  key: string
  label: string
  desc: string
  /** 对话首条意图消息（引擎按意图路由） */
  intent: string
  /** 特殊入口路由（如合盘 → /synastry），否则走对话意图 */
  route?: string
}

// 价值入口：按用户场景，不按引擎 skill。共用于工作台「发现+入口」与档案详情解读。
export const chatServices: ChatService[] = [
  { key: 'bazi', label: '八字紫微', desc: '命盘·大运·流年·格局', intent: '我想咨询八字。' },
  { key: 'naming', label: '起名', desc: '依八字用神取名，五行配字', intent: '我想咨询起名。' },
  { key: 'divination', label: '问卦', desc: '六爻·奇门断事', intent: '我想问卦。' },
  { key: 'fengshui', label: '风水', desc: '八宅·玄空·流年飞星', intent: '我想咨询风水。' },
  { key: 'bond', label: '合盘', desc: '两人八字合婚·关系分析', intent: '我想咨询合盘。', route: '/synastry' },
]

export function serviceByKey(key: string): ChatService | undefined {
  return chatServices.find(s => s.key === key)
}

export function serviceByIntent(intent: string): ChatService | undefined {
  return chatServices.find(s => s.intent === intent)
}
