import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

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

/** 黄历择日：start_date + count + 事项英文键，返回逐日事实数组。 */
export function engineHuangliDays(event: string, startDate: string, count: number): Promise<HuangliDay[]> {
  return mcpTool<HuangliDay[]>(ENGINE_TOOLS.HuangliDays, { event, start_date: startDate, count })
}

// ── bazi.liunian（流年盘）──
