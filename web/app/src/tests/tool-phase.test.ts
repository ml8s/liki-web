import { describe, expect, it } from 'vitest'
import { toolPhaseKey } from '@/utils/toolPhase'

describe('toolPhaseKey', () => {
  it.each([
    ['load_skill', 'chat.tool.intent'],
    ['load_skill_bazi', 'chat.tool.intent'],
    ['transfer_to_agent', 'chat.tool.intent'],
    ['bazi', 'chat.tool.bazi'],
    ['ziwei', 'chat.tool.ziwei'],
    ['city_coords', 'chat.tool.coords'],
    ['tianwen_resolve', 'chat.tool.coords'],
    ['bazi_chart', 'chat.tool.chart'],
    ['compute_factors', 'chat.tool.chart'],
    ['natal_query', 'chat.tool.chart'],
    ['period_query', 'chat.tool.period'],
    ['bazi_liunian', 'chat.tool.period'],
    ['qimen_cast', 'chat.tool.qimen'],
    ['liuyao_chart', 'chat.tool.liuyao'],
    ['naming_generate', 'chat.tool.naming'],
    ['fengshui_analyze', 'chat.tool.fengshui'],
    ['unknown_tool', 'chat.tool.generic'],
    ['', 'chat.tool.generic'],
  ])('%s -> %s', (tool, want) => {
    expect(toolPhaseKey(tool)).toBe(want)
  })
})
