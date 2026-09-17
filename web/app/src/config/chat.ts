export interface SkillDefinition {
  id: 'liki'
  label: string
  description: string
}

// 引擎已把四个领域合并为统一 skill「liki」，由引擎按意图自动路由；
// 前端保留一个通用 Chat 入口。
export const chatSkills: SkillDefinition[] = [
  {
    id: 'liki',
    label: '命理解读',
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

// 价值入口：按用户场景命名（看命/合盘/起名/问卦/风水），不按引擎 skill。
// route：每个入口都有免费盘面页（看命/风水→人列表排盘、起名/问卦/合盘→独立预览页）；付费解读统一进 Chat。
export const chatServices: ChatService[] = [
  { key: 'bazi', label: '看命', desc: '八字·紫微·大运·流年·格局', intent: '我想咨询八字。', route: '/profiles' },
  { key: 'naming', label: '起名', desc: '依八字用神取名，五行配字', intent: '我想咨询起名。', route: '/naming' },
  { key: 'divination', label: '问卦', desc: '六爻断事·用神应期', intent: '我想问卦。', route: '/divination' },
  { key: 'fengshui', label: '风水', desc: '八宅·流年飞星·吉凶方位', intent: '我想咨询风水。', route: '/fengshui' },
  { key: 'huangli', label: '择日', desc: '黄历择日·宜忌', intent: '我想咨询择日。', route: '/huangli' },
  { key: 'bond', label: '合盘', desc: '两人八字合婚·关系分析', intent: '我想咨询合盘。', route: '/synastry' },
]

export function serviceByKey(key: string): ChatService | undefined {
  return chatServices.find(s => s.key === key)
}

export function serviceByIntent(intent: string): ChatService | undefined {
  return chatServices.find(s => s.intent === intent)
}
