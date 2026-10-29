import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

/** Engine HuangliCount is schema-bounded; no silent clamping. */
export type HuangliCount = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10
  | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20
  | 21 | 22 | 23 | 24 | 25 | 26 | 27 | 28 | 29 | 30

/** 黄历单日（mcpcontract.HuangliDayOutput 线上形状）。 */
export interface HuangliShiChen {
  zhi: string
  time: string
  huangdao: string
  jian_chu: string
  suitable: boolean
}

export interface HuangliDay {
  date: string
  ri_zhu: { gan: string; zhi: string; na_yin: string }
  na_yin: string
  wuxing: string
  jian_chu: string
  huangdao: { index: number; name: string; path: string; sequence: number }
  xi_shen: string
  cai_shen: string
  fu_shen: string
  gan_ji: string
  zhi_ji: string
  mansion: { index: number; name: string; dong_wu: string; wuxing: string; group: string; group_idx: number }
  jie_qi: string
  jie_qi_days: number
  ren_yuan: string
  shi_chen?: HuangliShiChen[]
  event?: string
  event_label?: string
  suitability?: 'recommended' | 'possible' | 'unsuitable' | 'unspecified'
  reason?: string
}

/** 黄历择日：standard_time + count + 事项英文键，返回逐日事实数组。 */
export function engineHuangliDays(event: string, standardTime: string, count: HuangliCount): Promise<HuangliDay[]> {
  return mcpTool<{ items: HuangliDay[] }>(ENGINE_TOOLS.HuangliDays, {
    event,
    standard_time: standardTime,
    count,
  }).then(result => result.items)
}

// ── bazi.liunian（流年盘）──
