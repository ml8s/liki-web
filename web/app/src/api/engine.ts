// 免费排盘预览：浏览器调用 web BFF，BFF 服务端调用 engine-mcp（只读，不落库）。
// MCP 工具契约归 liki/engine；下游 token 永不进入浏览器。
import { ApiError } from './http'

interface EngineToolEnvelope<T> {
  data?: T
  detail?: string
  /** RFC 9457 problem 的 type：BFF 机器码（engine_tool_failed / engine_tool_not_found 等）。 */
  type?: string
}

/** 免费 Web UI 可用的 engine MCP 工具名（API 层和 E2E mock 共用）。 */
export const ENGINE_TOOLS = {
  TianwenTime: 'tianwen_time',
  CityCoords: 'city_coords',
  BaziChart: 'bazi_chart',
  BaziFullChart: 'bazi_fullchart',
  ZiweiChart: 'ziwei_chart',
  ZiweiFullChart: 'ziwei_fullchart',
  BazhaiChart: 'bazhai_chart',
  BaziBond: 'bazi_bond',
  LiuyaoQigua: 'liuyao_qigua',
  LiuyaoChart: 'liuyao_chart',
  HuangliDays: 'huangli_days',
} as const

export async function engineTool<T = unknown>(tool: string, params: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch(`/api/engine/tools/${encodeURIComponent(tool)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Network request failed')
  }
  let envelope: EngineToolEnvelope<T>
  try {
    envelope = await response.json() as EngineToolEnvelope<T>
  } catch {
    if (!response.ok) {
      throw new ApiError(response.status, 'engine_unavailable', `Engine HTTP ${response.status}`)
    }
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  if (!response.ok) {
    // BFF 错误按 RFC 9457 返回 problem+json，type 即机器码；直接作为 code 透传。
    throw new ApiError(response.status, envelope.type || 'engine_unavailable', envelope.detail || `Engine HTTP ${response.status}`)
  }
  if (envelope.detail) {
    throw new ApiError(response.status, 'engine_tool_failed', envelope.detail)
  }
  if (!envelope || envelope.data === undefined) {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  return envelope.data
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
  return engineTool<{ solar: string; lunar: string }>(ENGINE_TOOLS.TianwenTime, { time, longitude })
}

export function engineCityCoords(city: string): Promise<{ longitude: number; latitude: number }> {
  return engineTool<{ longitude: number; latitude: number }>(ENGINE_TOOLS.CityCoords, { city })
}

export function engineBaziChart(solarTime: string, gender: string): Promise<BaziChartData> {
  return engineTool<BaziChartData>(ENGINE_TOOLS.BaziChart, { solar_time: solarTime, gender })
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
  return engineTool<BaziFullChartData>(ENGINE_TOOLS.BaziFullChart, { chart })
}

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

export type ZiweiChartData = ZiweiChart

export function engineZiweiChart(lunar: string, gender: string): Promise<ZiweiChartData> {
  return engineTool<ZiweiChartData>(ENGINE_TOOLS.ZiweiChart, { lunar, gender })
}

export function engineZiweiFullChart(chart: unknown): Promise<ZiweiChart> {
  return engineTool<ZiweiChart>(ENGINE_TOOLS.ZiweiFullChart, { chart })
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
  return engineTool<BazhaiChart>(ENGINE_TOOLS.BazhaiChart, { birth_year: birthYear, gender })
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
  a_zhu: string
  b_zhu: string
  a_gan: string
  b_gan: string
  a_zhi: string
  b_zhi: string
  gan_guan_xi: BaziGanRelation
  zhi_guan_xi: BaziZhiRelation
}

export interface BaziBondNayinPair {
  a_zhu: string
  b_zhu: string
  a_na_yin: string
  b_na_yin: string
  relation: string
}

export interface BaziBondResult {
  zhu_cross?: { pairs: BaziBondPair[] }
  shi_shen_cross?: { a_to_b: Record<string, string>; b_to_a: Record<string, string> }
  nayin_cross?: { pairs: BaziBondNayinPair[] }
  shensha_cross?: Record<string, { a_present_in_b: boolean; b_present_in_a: boolean }>
}

/** 八字合盘：输入双方最小命盘（bazi.chart 结果）。 */
export function engineBaziBond(chartA: unknown, chartB: unknown): Promise<BaziBondResult> {
  return engineTool<BaziBondResult>(ENGINE_TOOLS.BaziBond, { a: { chart: chartA }, b: { chart: chartB } })
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
  return engineTool<{ casting: LiuyaoCasting }>(ENGINE_TOOLS.LiuyaoQigua, { mode })
}

/** 六爻装卦：传入 casting + 真太阳时，返回卦象/用神/应期。 */
export function engineLiuyaoChart(solarTime: string, casting: LiuyaoCasting, yongShen?: string): Promise<LiuyaoChartResult> {
  const params: Record<string, unknown> = { solar_time: solarTime, casting }
  if (yongShen) params.yong_shen = yongShen
  return engineTool<LiuyaoChartResult>(ENGINE_TOOLS.LiuyaoChart, params)
}

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

/** 黄历择日：start_date + count（引擎上限 30 天）+ 事项英文键，返回逐日事实数组。 */
export function engineHuangliDays(event: string, startDate: string, count: number): Promise<HuangliDay[]> {
  return engineTool<HuangliDay[]>(ENGINE_TOOLS.HuangliDays, { event, start_date: startDate, count })
}
