import type { ChartInput } from '../../domain/time'
import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

export interface LiuyaoCasting {
  mode: LiuyaoQiguaMode
  order: 'bottom_up'
  coin_convention?: { 正: 3; 反: 2 }
  rounds?: SixItems<LiuyaoCoinRound>
  yaos: SixItems<LiuyaoYaoValue>
  dong_yao: LiuyaoPosition[]
  casting_id: string
}

export interface LiuyaoLine {
  position?: number
  type?: LiuyaoYaoValue
  gan?: string
  zhi?: string
  wuxing?: string
  liu_qin?: string
  shi_ying?: string
  liu_shou?: string
  wang_shuai?: string
  yue_po?: boolean
  dong_self?: boolean
  dong_sheng?: boolean
  dong_ke?: boolean
  xun_kong?: boolean
  mu_ku?: boolean
}

export interface LiuyaoYongShen {
  name?: string
  position?: number
  is_hidden?: boolean
  wang_shuai?: string
  yue_po?: boolean
  xun_kong?: boolean
  mu_ku?: boolean
  liu_shou?: string
  chang_sheng?: string
}

export interface LiuyaoTimingCandidate {
  id: string
  description?: string
  mechanism: string
  position?: number
  trigger_branch?: string
  window?: string
  basis?: string[]
  confidence: string
}

export interface LiuyaoPattern {
  type: string
  sub_type: string
  position: number
  is_true: boolean
}

export interface LiuyaoChartResult {
  name: string
  ben_gua: string
  bian_gua?: string
  gong: string
  gong_wuxing: string
  lines: SixItems<LiuyaoLine>
  bian_yao: SixItems<LiuyaoLine>
  ri_chen_gan: string
  ri_chen_zhi: string
  yue_jian_zhi: string
  yue_jian_gan: string
  yong_shen: LiuyaoYongShen
  gua_ci: { name: string; gua_ci: string; yao_ci: SixItems<string> }
  wang_shuai: SixItems<string>
  ri_chen_relations: SixItems<{ relations: string[] }>
  xun_kong: [string, string]
  timing_candidates?: LiuyaoTimingCandidate[]
  patterns?: LiuyaoPattern[]
  casting: LiuyaoCasting
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
