// 免费排盘预览：前端 TS 直连引擎 RPC（只读消费，不落库）。
// 引擎 RPC 契约归 liki/engine，方法名/入参以引擎现状为准，绝不修改。
import { ApiError } from './http'

let rpcId = 0

interface RpcEnvelope<T> {
  result?: T
  error?: { code: number; message: string }
}

interface RpcResult<T> {
  data: T
}

/** 引擎 RPC 方法名（单一真相源——API 层和 E2E mock 均从此导入）。 */
export const RPC_METHODS = {
  TianwenTime: 'tianwen.time',
  CityCoords: 'city.coords',
  BaziChart: 'bazi.chart',
  BaziFullChart: 'bazi.fullchart',
  ZiweiChart: 'ziwei.chart',
  ZiweiFullChart: 'ziwei.fullchart',
  BazhaiChart: 'bazhai.chart',
  BaziBond: 'bazi.bond',
  LiuyaoQigua: 'liuyao.qigua',
  LiuyaoChart: 'liuyao.chart',
  HuangliDays: 'huangli.days',
} as const

export async function engineRpc<T = unknown>(method: string, params: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch('/jsonrpc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ jsonrpc: '2.0', method, params, id: (rpcId = (rpcId + 1) % 1_000_000) }),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Network request failed')
  }
  if (!response.ok) {
    throw new ApiError(response.status, 'engine_unavailable', `Engine HTTP ${response.status}`)
  }
  let envelope: RpcEnvelope<RpcResult<T>>
  try {
    envelope = await response.json() as RpcEnvelope<RpcResult<T>>
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  if (envelope.error) {
    throw new ApiError(response.status, 'engine_rpc_failed', envelope.error.message)
  }
  if (!envelope.result || !('data' in envelope.result)) {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  return envelope.result.data
}

export interface BaziZhu {
  gan: string
  zhi: string
  na_yin?: string
}

export interface BaziDaYunRootRef {
  source: string
  pillar?: string
  stem: string
  branch: string
}

export interface BaziDaYunStep {
  gan: string
  zhi: string
  name: string
  start_date?: string
  end_date?: string
  start_year?: number
  end_year?: number
  wuxing?: string
  shi_shen?: string
  rooted?: boolean
  root_refs?: BaziDaYunRootRef[]
}

export interface BaziDaYun {
  steps: BaziDaYunStep[]
  current_step_index: number
  next_step_in_years?: number
  direction?: string
}

export interface BaziChartData {
  nian: BaziZhu
  yue: BaziZhu
  ri: BaziZhu
  shi: BaziZhu
  birth_year?: number
  gender?: string
  zi_shi_rule?: string
  zodiac?: string
  day_xun?: string
  day_xun_kong?: string
  da_yun?: BaziDaYun
}

export function engineSolarTime(time: string, longitude: number): Promise<{ solar: string; lunar: string }> {
  return engineRpc<{ solar: string; lunar: string }>(RPC_METHODS.TianwenTime, { time, longitude })
}

export function engineCityCoords(city: string): Promise<{ longitude: number; latitude: number }> {
  return engineRpc<{ longitude: number; latitude: number }>(RPC_METHODS.CityCoords, { city })
}

export function engineBaziChart(solarTime: string, gender: string): Promise<BaziChartData> {
  return engineRpc<BaziChartData>(RPC_METHODS.BaziChart, { solar_time: solarTime, gender })
}

export interface BaziFuYiBasis {
  root_type?: string
  season?: string
  yin_bi_count?: number
  day_master_roots?: Array<{ stem: string; zhi: string }>
  relation_facts?: Array<{ field: string; group: string; pillars?: string[]; branches?: string[]; targets?: string[] }>
}

export interface BaziFuYi {
  qiangruo?: string
  yong?: string
  xi?: string
  ji?: string
  wuxing_count?: Record<string, number>
  wang_shuai?: Record<string, string>
  pattern?: string
  model?: string
  basis?: BaziFuYiBasis
}

export interface BaziTiaoHouAvailability {
  stem?: string
  transparent?: boolean
  hidden?: boolean
  branch?: string
}

export interface BaziTiaoHou {
  yong?: string
  xi?: string
  season?: string
  detail?: string
  model?: string
  primary?: BaziTiaoHouAvailability
  secondary?: BaziTiaoHouAvailability
}

export interface BaziGeJuStructure {
  wuxing?: string
  transparent_pillars?: string[]
  roots?: Array<{ stem: string; zhi: string }>
  season?: string
  timely?: boolean
  strength?: string
}

export interface BaziGeJu {
  ge_ju?: string
  yong_fa?: string
  yong?: string
  xi?: string
  ji?: string
  pattern_god?: string
  pattern_god_ten_god?: string
  pattern_god_source?: string
  structure?: {
    pattern?: BaziGeJuStructure
    controller?: BaziGeJuStructure
    generator?: BaziGeJuStructure
    relation_facts?: Array<{ field: string; group: string; pillars?: string[]; branches?: string[]; targets?: string[] }>
  }
}

export interface BaziYongShen {
  fu_yi?: BaziFuYi
  tiao_hou?: BaziTiaoHou
  ge_ju?: BaziGeJu
}

export interface BaziShenSha {
  name: string
  category?: string
  description?: string
}

export interface BaziShiShenEntry {
  shi_shen: string
  name?: string
  source?: string
  gan?: string
}

export interface BaziZhuFull extends BaziZhu {
  day_master_trend?: string
  xun?: string
  xun_kong?: string
  self_sitting?: string
  shi_shens?: BaziShiShenEntry[]
  cang_gan?: { main?: string; mid?: string; minor?: string }
  shen_sha?: BaziShenSha[]
  is_void?: boolean
  is_self_he?: boolean
  is_kui_gang?: boolean
  self_he_name?: string
}

export interface BaziSanYuan {
  tai_yuan?: { gan: string; zhi: string }
  ming_gong?: { gan: string; zhi: string }
  shen_gong?: { gan: string; zhi: string }
}

export interface ChangShengStage {
  name: string
  index: string
}

export interface TenGodState {
  shi_shen: string
  wuxing: string
  transparent: boolean
  hidden: boolean
  rooted: boolean
  timely: boolean
  count: number
  strength: string
}

export interface ElementState {
  wuxing: string
  season: string
  season_strength: string
  strength: string
  transparent: boolean
  rooted: boolean
  controls: string
  controlled_by: string
  controller_strength: string
  reasons: string[]
}

export interface RelationGroup {
  field: string
  group: string
}

export interface LuRoot {
  shi_shen: string
  gan: string
  zhi: string
}

export interface NayinRelEntry {
  a: string
  b: string
  relation: string
}

export interface GongJia {
  pillar_a: number
  pillar_b: number
  type: string
  zhi: string
}

export interface AtomicFacts {
  day_master_element?: string
  day_master_stem?: string
  day_branch?: string
  month_longevity?: string
  year_stem_ten_god?: string
  month_main_ten_god?: string
  hour_stem_ten_god?: string
  pattern_god_transparent?: boolean
  pillar_punishments?: Record<string, boolean>
  officer_killing_cleaned?: boolean
  wealth_tomb_present?: boolean
  wealth_star_in_tomb?: boolean
}

export interface ShenShaSchool {
  dual_reference?: string[]
  policy?: string
}

export interface BaziFullChartData {
  nian: BaziZhuFull
  yue: BaziZhuFull
  ri: BaziZhuFull
  shi: BaziZhuFull
  da_yun?: BaziDaYun
  gender?: string
  birth_year?: number
  yong_shen?: BaziYongShen
  fu_yi?: BaziFuYi
  tiao_hou?: BaziTiaoHou
  ge_ju?: BaziGeJu
  shen_sha_school?: ShenShaSchool
  lu_roots?: LuRoot[]
  relation_groups?: RelationGroup[]
  ten_god_states?: TenGodState[]
  element_states?: ElementState[]
  atomic_facts?: AtomicFacts
  san_yuan?: BaziSanYuan
  tai_xi?: { gan: string; zhi: string }
  gong_jia?: GongJia[]
  nayin_rel?: NayinRelEntry[]
  chang_sheng?: ChangShengStage[]
  san_qi_name?: string
  zodiac?: string
  gan_he?: Array<{ gan_a: string; gan_b: string; type: string; relation: string }>
  zhi_liu_he?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  san_he?: Array<{ field: string; group: string }>
  san_hui?: Array<{ field: string; group: string }>
  liu_chong?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  liu_hai?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  liu_xing?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  liu_po?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  an_he?: Array<{ zhi_a: string; zhi_b: string; type: string; detail: string }>
  day_xun?: string
  day_xun_kong?: string
}

export function engineBaziFullChart(chart: unknown): Promise<BaziFullChartData> {
  return engineRpc<BaziFullChartData>(RPC_METHODS.BaziFullChart, { chart })
}

export interface ZiweiStarInfo {
  xing: string
  name: string
  is_zhu_xing?: boolean
  si_hua?: string
  liang_du?: string
}

export interface ZiweiGong {
  index: number
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

export interface ZiweiChart {
  school?: string
  gong_wei: ZiweiGong[]
  ming_gong?: number
  shen_gong?: number
  ju_shu?: number
  ju_shu_name?: string
  ziwei_pos?: number
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

export interface ZiweiChartData {
  school?: string
  gong_wei: ZiweiGong[]
  ming_gong?: number
  shen_gong?: number
  ju_shu?: number
  ju_shu_name?: string
  ziwei_pos?: number
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

export function engineZiweiChart(lunar: string, gender: string): Promise<ZiweiChartData> {
  return engineRpc<ZiweiChartData>(RPC_METHODS.ZiweiChart, { lunar, gender })
}

export function engineZiweiFullChart(chart: unknown): Promise<ZiweiChart> {
  return engineRpc<ZiweiChart>(RPC_METHODS.ZiweiFullChart, { chart })
}

export interface BazhaiLiuNianXing {
  year?: number
  ru_zhong?: string
  gong_wei?: Array<{ gong_num: number; xing_name: string; wuxing?: string; ru_zhong?: boolean }>
}

export interface BazhaiChart {
  ming_gua: { gua: { name: string; wuxing?: string }; group: string }
  ba_zhai_dirs: Record<string, string[]>
  liu_nian_xing?: BazhaiLiuNianXing
}

export function engineBazhaiChart(birthYear: number, gender: string): Promise<BazhaiChart> {
  return engineRpc<BazhaiChart>(RPC_METHODS.BazhaiChart, { birth_year: birthYear, gender })
}

export interface BaziGanRelation {
  gan_a: string
  gan_b: string
  type: string
  relation: string
}

export interface BaziZhiRelation {
  zhi_a: string
  zhi_b: string
  type: string
  detail: string
}

export interface BaziBondPair {
  /** 柱位键：nian/yue/ri/shi */
  jia_zhu: string
  yi_zhu: string
  jia_gan: string
  yi_gan: string
  jia_zhi: string
  yi_zhi: string
  gan_guan_xi: BaziGanRelation
  zhi_guan_xi: BaziZhiRelation
}

export interface BaziBondNayinPair {
  jia_zhu: string
  yi_zhu: string
  jia_na_yin: string
  yi_na_yin: string
  relation: string
}

export interface BaziBondResult {
  zhu_cross?: { pairs: BaziBondPair[] }
  shi_shen_cross?: { jia_dui_yi: Record<string, string>; yi_dui_jia: Record<string, string> }
  nayin_cross?: { pairs: BaziBondNayinPair[] }
  shensha_cross?: Record<string, { jia_ru_yi: boolean; yi_ru_jia: boolean }>
  structure?: unknown
}

/** 八字合盘：输入双方最小命盘（bazi.chart 结果）。 */
export function engineBaziBond(chartA: unknown, chartB: unknown): Promise<BaziBondResult> {
  return engineRpc<BaziBondResult>(RPC_METHODS.BaziBond, { a: { chart: chartA }, b: { chart: chartB } })
}

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

/** 六爻起卦：生成 casting 收据。 */
export function engineLiuyaoQigua(mode = 'auto'): Promise<{ casting: LiuyaoCasting }> {
  return engineRpc<{ casting: LiuyaoCasting }>(RPC_METHODS.LiuyaoQigua, { mode })
}

/** 六爻装卦：传入 casting + 真太阳时，返回卦象/用神/应期。 */
export function engineLiuyaoChart(solarTime: string, casting: LiuyaoCasting, yongShen?: string): Promise<LiuyaoChartResult> {
  const params: Record<string, unknown> = { solar_time: solarTime, casting }
  if (yongShen) params.yong_shen = yongShen
  return engineRpc<LiuyaoChartResult>(RPC_METHODS.LiuyaoChart, params)
}

export interface HuangliCandidate {
  date: string
  jian_chu?: string
  suitability?: 'recommended' | 'possible' | 'unsuitable' | 'unspecified'
  reason?: string
  warnings?: string[]
}

export interface HuangliDaysResult {
  event?: string
  event_label?: string
  range?: { start_date: string; end_date: string; days: number }
  recommended?: HuangliCandidate[]
  unsuitable?: HuangliCandidate[]
  candidates?: HuangliCandidate[]
  safety_advisory?: string
}

/** 黄历择日：按事项与日期范围返回推荐/排除候选。 */
export function engineHuangliDays(question: string, event: string, startDate: string, endDate: string): Promise<HuangliDaysResult> {
  return engineRpc<HuangliDaysResult>(RPC_METHODS.HuangliDays, { question, event, start_date: startDate, end_date: endDate })
}
