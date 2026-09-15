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

// 产品包含的领域（用于购买卡展示，非 skill 标识）。
export const chatDomains = ['八字 · 紫微', '起名 · 改名', '六爻 · 奇门', '八宅 · 玄空']
