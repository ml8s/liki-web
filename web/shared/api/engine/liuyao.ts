import type { ChartInput } from '../../domain/time'
import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

export interface LiuyaoCasting {
  schema_version?: string
  mode?: string
  order?: number[]
  yaos?: unknown[]
  dong_yao?: number[]
  casting_id?: string
  rounds?: unknown[]
}

export interface LiuyaoLine {
  xing?: string
  liu_qin?: string
  di_zhi?: string
  tian_gan?: string
  wang_shuai?: string
  ri_chen_relation?: string
}

export interface LiuyaoBoard {
  name?: string
  ben_gua?: string
  bian_gua?: string
  gong?: string
  gong_wuxing?: string
  ri_chen_gan?: string
  ri_chen_zhi?: string
  yue_jian_zhi?: string
  yue_jian_gan?: string
  xun_kong?: string[]
  lines?: LiuyaoLine[]
  bian_yao?: LiuyaoLine[]
}

export interface LiuyaoYongShen {
  name?: string
  position?: number
  world?: string
  other?: string
}

export interface LiuyaoTimingCandidate {
  time?: string
  description?: string
}

export interface LiuyaoPattern {
  name?: string
  description?: string
}

export interface LiuyaoChartResult {
  casting?: LiuyaoCasting
  board?: LiuyaoBoard
  focus?: { yong_shen?: string; world?: string; other?: string }
  yong_shen?: LiuyaoYongShen
  gua_ci?: { gua?: string; ci?: string }
  timing_candidates?: LiuyaoTimingCandidate[]
  safety_advisory?: string
  patterns?: LiuyaoPattern[]
}

export type LiuyaoPosition = 1 | 2 | 3 | 4 | 5 | 6
export type LiuyaoCoinSymbol = '正' | '反'
export type LiuyaoYaoValue = 6 | 7 | 8 | 9
export type LiuyaoYaoLabel = '老阴' | '少阳' | '少阴' | '老阳'

export interface LiuyaoCoinRound {
  position: LiuyaoPosition
  coins: [LiuyaoCoinSymbol, LiuyaoCoinSymbol, LiuyaoCoinSymbol]
  value: LiuyaoYaoValue
  label: LiuyaoYaoLabel
  changing: boolean
}

type SixItems<T> = [T, T, T, T, T, T]

/** Engine mcpcontract.LiuyaoQiguaMode: auto / coins / yaos. */
export type LiuyaoQiguaMode = 'auto' | 'coins' | 'yaos'
export type LiuyaoQiguaInput =
  | { mode: 'auto' }
  | { mode: 'coins'; rounds: SixItems<LiuyaoCoinRound> }
  | { mode: 'yaos'; yaos: SixItems<LiuyaoYaoValue> }

/** 六爻起卦：生成 casting 收据。 */
export function engineLiuyaoQigua(
  input: LiuyaoQiguaInput,
): Promise<{ casting: LiuyaoCasting }> {
  return mcpTool<{ casting: LiuyaoCasting }>(ENGINE_TOOLS.LiuyaoQigua, input)
}

/** 六爻装卦：传入 casting + 真太阳时，返回卦象/用神/应期。 */
export function engineLiuyaoChart(
  input: ChartInput & { casting: LiuyaoCasting; yongShen?: string },
): Promise<LiuyaoChartResult> {
  const params: Record<string, unknown> = {
    instant: input.instant,
    timezone: input.timezone,
    longitude: input.longitude,
    casting: input.casting,
  }
  if (input.yongShen) params.yong_shen = input.yongShen
  return mcpTool<LiuyaoChartResult>(ENGINE_TOOLS.LiuyaoChart, params)
}
