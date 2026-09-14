export interface SkillDefinition {
  id: 'liki-naming' | 'liki-bazi' | 'liki-divination' | 'liki-fengshui'
  label: string
  description: string
}

export const chatSkills: SkillDefinition[] = [
  {
    id: 'liki-naming',
    label: '起名',
    description: '中文名、改名、名字评估与外国人起名。',
  },
  {
    id: 'liki-bazi',
    label: '八字紫微',
    description: '命盘、大运流年、格局与喜忌分析。',
  },
  {
    id: 'liki-divination',
    label: '问卦',
    description: '六爻、奇门、黄历与具体事项判断。',
  },
  {
    id: 'liki-fengshui',
    label: '风水',
    description: '阳宅坐向、八宅与玄空布局分析。',
  },
]
