import type { ChartInput } from '../../domain/time'
import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

export interface ZiweiStarInfo {
  xing: string
  name: string
  is_zhu_xing?: boolean
  si_hua?: string
  liang_du?: string
}

export interface ZiweiGong {
  /** 与 name 同值的宫名（引擎 wire 如 "命宫"）。 */
  index: string
  name: string
  gan?: string
  zhi?: string
  is_shen_gong?: boolean
  is_yuan_gong?: boolean
  xing_yao: ZiweiStarInfo[]
  zi_wei?: number
  ages?: number[]
  chang_sheng?: string
  bo_shi?: string
  jiang_qian?: string
  sui_qian?: string
  za_yao?: string[]
}

export interface ZiweiSiHuaResult {
  [star: string]: string
}

export interface ZiweiKongGong {
  gong_name: string
  jie_xing: string[]
  dui_gong: string
}

export interface ZiweiPattern {
  name: string
  description: string
}

export interface ZiweiSanFang {
  name: string
  zhu_xing: string[]
  fu_xing: string[]
  si_hua: string[]
}

export interface ZiweiPalaceFact {
  palace: string
  kind: string
  target: string
  star?: string
  value?: string
}

/** 紫微命盘（ziwei_chart / ziwei_fullchart 共用同一 wire DTO）。 */
export interface ZiweiChart {
  school?: { leap_month?: string; source?: string }
  gong_wei: ZiweiGong[]
  ming_gong?: string
  shen_gong?: string
  ju_shu?: string
  ju_shu_name?: string
  ziwei_pos?: string
  si_hua?: ZiweiSiHuaResult
  palace_facts?: ZiweiPalaceFact[]
  kong_gong?: ZiweiKongGong[]
  nian_gan?: string
  nian_zhi?: string
  shi_zhi?: string
  birth_year?: number
  lunar_month?: number
  lunar_day?: number
  birth_lunar_month?: number
  birth_is_leap?: boolean
  gender?: string
  ming_zhu?: string
  shen_zhu?: string
  patterns?: ZiweiPattern[]
  san_fang?: ZiweiSanFang[]
}

export function engineZiweiChart(input: ChartInput & { gender: string }): Promise<ZiweiChart> {
  return mcpTool<ZiweiChart>(ENGINE_TOOLS.ZiweiChart, {
    instant: input.instant,
    timezone: input.timezone,
    longitude: input.longitude,
    gender: input.gender,
  })
}

export function engineZiweiFullChart(chart: unknown): Promise<ZiweiChart> {
  return mcpTool<ZiweiChart>(ENGINE_TOOLS.ZiweiFullChart, { chart })
}

/** 大限（ziwei.daxian 返回数组根）。 */
export interface ZiweiDaXianItem {
  name: string
  gong: string
  qi_sui: number
  zhi_sui: number
  start_year?: number
  end_year?: number
}

export function engineZiweiDaXian(chart: unknown): Promise<ZiweiDaXianItem[]> {
  return mcpTool<ZiweiDaXianItem[]>(ENGINE_TOOLS.ZiweiDaXian, { chart })
}

/** 流年十二宫（ziwei.liunian；gong_wei 为流年盘宫名，ming_gong_name 为流年命宫落本命宫）。 */
export interface ZiweiLiuNianGong {
  zhi: string
  name: string
  xing_yao?: string[]
  is_liu_ming?: boolean
}

export interface ZiweiLiuNianData {
  ming_gong: string
  ming_gong_name?: string
  zhi: string
  si_hua?: Record<string, string>
  si_hua_gong?: Record<string, string>
  fu_xing?: Record<string, string>
  gong_wei?: ZiweiLiuNianGong[]
}

export function engineZiweiLiuNian(chart: unknown, lunarYear: number): Promise<ZiweiLiuNianData> {
  return mcpTool<ZiweiLiuNianData>(ENGINE_TOOLS.ZiweiLiuNian, { chart, lunar_year: lunarYear })
}
