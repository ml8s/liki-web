import type { ChartInput } from '../../domain/time'
import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

export type BaziElement = '木' | '火' | '土' | '金' | '水'

/** Engine/counsel publish candidate absence as explicit null, not "". */
export type BaziElementCandidate = BaziElement | null

export interface BaziZhu {
  gan: string
  zhi: string
  na_yin: string
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
  start_date: string
  end_date: string
  start_year: number
  end_year: number
  wuxing: string
  shi_shen: string
  rooted?: boolean
  root_refs?: BaziDaYunRootRef[]
}

export interface BaziDaYun {
  start_date: string
  start_year_after: number
  start_month_after: number
  start_day_after: number
  direction: string
  steps: BaziDaYunStep[]
}

export interface BaziChartData {
  nian: BaziZhu
  yue: BaziZhu
  ri: BaziZhu
  shi: BaziZhu
  birth_year: number
  gender: 'male' | 'female'
  zi_shi_rule: string
  da_yun?: BaziDaYun
}

export function engineBaziChart(input: ChartInput & { gender: string }): Promise<BaziChartData> {
  return mcpTool<BaziChartData>(ENGINE_TOOLS.BaziChart, {
    standard_time: input.standard_time,
    longitude: input.longitude,
    gender: input.gender,
  })
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
  yong: BaziElementCandidate
  xi: BaziElementCandidate
  ji: BaziElementCandidate
  wuxing_count?: Record<string, number>
  wang_shuai?: Record<string, string>
  pattern?: string
  model?: string
  basis?: BaziFuYiBasis
}

export interface BaziTiaoHouCandidate {
  stem?: string
  transparent?: boolean
  hidden?: boolean
  occurrences?: Array<{ branch?: string; pillar?: string; source?: string; stem?: string }>
  relation_facts?: Array<{
    branches?: string[]
    field?: string
    group?: string
    pillars?: string[]
    targets?: string[]
  }>
}

export interface BaziTiaoHouAvailability {
  stem?: string
  transparent?: boolean
  hidden?: boolean
  branch?: string
}

export interface BaziTiaoHou {
  primary_wuxing: BaziElement
  secondary_wuxing: BaziElementCandidate
  season?: string
  detail?: string
  model?: string
  primary?: BaziTiaoHouAvailability
  secondary?: BaziTiaoHouAvailability
  secondary_condition?: string | null
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
  /** 无自合为 null。 */
  self_he_name?: string | null
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

/** 拱夹：两柱之间拱出的地支（wire：pillar_a/pillar_b 为柱位下标 0-3）。 */
export interface GongJia {
  pillar_a: number
  pillar_b: number
  type: string
  zhi: string
  wuxing?: string
}

/** 天干五合（wire 实测：contested=争合，he_element=合化五行）。 */
export interface BaziGanHe {
  gan_a: string
  gan_b: string
  pillar_a: number
  pillar_b: number
  he_element?: string
  position?: string
  contested?: boolean
}

/** 天干相冲。 */
export interface BaziGanChong {
  gan_a: string
  gan_b: string
  pillar_a: number
  pillar_b: number
  position?: string
}

/** 地支六合（wuxing=合化五行）。 */
export interface BaziZhiLiuHe {
  zhi_a: string
  zhi_b: string
  pillar_a: number
  pillar_b: number
  wuxing?: string
}

/** 刑冲害破暗合等纯地支对（wire 实测仅柱位与地支）。 */
export interface BaziZhiPair {
  zhi_a: string
  zhi_b: string
  pillar_a: number
  pillar_b: number
}

/** 合局（三合/半合/三会）：name 为成局名（如「子辰水半合局」），pillars 为柱位键。 */
export interface BaziJuRelation {
  type?: string
  name?: string
  wuxing?: string
  branches?: string[]
  pillars?: string[]
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
  gan_he?: BaziGanHe[]
  gan_chong?: BaziGanChong[]
  zhi_liu_he?: BaziZhiLiuHe[]
  san_he?: BaziJuRelation[]
  san_he_partial?: BaziJuRelation[]
  san_hui?: BaziJuRelation[]
  liu_chong?: BaziZhiPair[]
  liu_hai?: BaziZhiPair[]
  liu_xing?: BaziZhiPair[]
  liu_po?: BaziZhiPair[]
  an_he?: BaziZhiPair[]
  day_xun?: string
  day_xun_kong?: string
}

export function engineBaziFullChart(chart: unknown): Promise<BaziFullChartData> {
  return mcpTool<BaziFullChartData>(ENGINE_TOOLS.BaziFullChart, { chart })
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
  day_master: {
    a: { gan: string; gender: string; wuxing: string }
    b: { gan: string; gender: string; wuxing: string }
    gan_guan_xi: BaziGanRelation
    b_present_in_a: boolean
  }
  spouse_palace: Record<string, unknown>
  zhu_cross: { pairs: BaziBondPair[] }
  shi_shen_cross: { a_to_b: Record<string, string>; b_to_a: Record<string, string> }
  wuxing_cross: Record<string, unknown>
  nayin_cross: { pairs: BaziBondNayinPair[] }
  spouse_star: Record<string, unknown>
  shensha_cross: Record<string, { a_present_in_b: boolean; b_present_in_a: boolean }>
}

/** 八字合盘：输入双方最小命盘（bazi_chart 结果）。 */
export function engineBaziBond(chartA: unknown, chartB: unknown): Promise<BaziBondResult> {
  return mcpTool<BaziBondResult>(ENGINE_TOOLS.BaziBond, { a: { chart: chartA }, b: { chart: chartB } })
}

export interface BaziYongShenResult {
  selected: {
    yong: BaziElementCandidate
    xi: string[]
    ji: string[]
    /** 判据来源：tiao_hou / fu_yi / ge_ju … */
    source: string
  }
  status: 'resolved' | 'partial' | 'needs_review' | 'unresolved'
  conflicts: Array<{ code: string; message: string }>
  rule_id: string
  policy_id: string
  chart_digest: string
  yongshen_digest: string
  ge_ju_context: {
    ge_ju: string
    pattern_god_ten_god: string
    pattern_god_source: string
    yong_fa: string
    clear: boolean
    theme: string
  }
  evidence: Array<{ source: 'facts' | 'fu_yi' | 'tiao_hou' | 'ge_ju' | 'cong_ge' } & Record<string, unknown>>
}

/** 起名/择吉的用神裁决（counsel 域，直连 /counsel/mcp）。 */
export function engineBaziYongShen(chart: BaziChartData): Promise<BaziYongShenResult> {
  return mcpTool<BaziYongShenResult>('bazi_yongshen', { chart })
}

export interface BaziLiuNianShensha {
  name: string
  category: string
  description?: string
}

export interface BaziLiuNianFuyinFanyin {
  type?: string
  detail?: string
  natal_index?: number
}

export interface BaziLiuNianGanRel {
  gan_a?: string
  gan_b?: string
  relation?: string
  type?: string
}

export interface BaziLiuNianZhiRel {
  zhi_a?: string
  zhi_b?: string
  detail?: string
  type?: string
}

/** 行运关系：一条柱（流年/大运）对命局四柱的干支关系。 */
export interface BaziLiuNianInteraction {
  pillar_label?: string
  gan_rels?: BaziLiuNianGanRel[]
  zhi_rels?: BaziLiuNianZhiRel[]
}

export interface BaziLiuNianOutput {
  year: number
  nian_name: string
  shi_shen: string
  nian_gan?: string
  nian_zhi?: string
  wuxing?: string
  na_yin?: string
  /** 流年五行对日主的生数/克数（引擎 countGenRest）。 */
  sheng?: number
  ke?: number
  shensha?: BaziLiuNianShensha[]
  fuyin_fanyin?: BaziLiuNianFuyinFanyin[]
  dayun_interactions?: BaziLiuNianInteraction[]
  natal_interactions?: BaziLiuNianInteraction[]
}

/** 流年盘（bazi.liunian）：最小命盘 + 目标年标准时间。 */
export function engineBaziLiunian(
  chart: unknown,
  standardTime: string,
  count = 1,
): Promise<BaziLiuNianOutput[]> {
  return mcpTool<{ items: BaziLiuNianOutput[] }>(ENGINE_TOOLS.BaziLiuNian, {
    standard_time: standardTime,
    count,
    chart,
  }).then(result => result.items)
}

export interface BaziLiuRiResolved {
  standard_time: string
  local_date: string
  last_date: string
  beijing_date: string
}

export interface BaziLiuRiShensha {
  name: string
  category: string
  description?: string
}

export interface BaziLiuRiGanRel {
  gan_a?: string
  gan_b?: string
  relation?: string
  type?: string
}

export interface BaziLiuRiZhiRel {
  detail?: string
  type?: string
  zhi_a?: string
  zhi_b?: string
}

export interface BaziLiuRiOutput {
  date: string
  day_name: string
  shi_shen: string
  ri_gan?: string
  ri_zhi?: string
  day_nayin?: string
  shensha?: BaziLiuRiShensha[]
  gan_rels?: BaziLiuRiGanRel[]
  zhi_rels?: BaziLiuRiZhiRel[]
  dayun_rels?: BaziLiuRiZhiRel[]
  liunian_rels?: BaziLiuRiZhiRel[]
  resolved?: BaziLiuRiResolved
}

/** 流日盘（bazi.liuri）：最小命盘 + 目标日标准时间与真太阳时经度。 */
export function engineBaziLiuri(
  chart: unknown,
  standardTime: string,
  longitude: number,
  count = 1,
): Promise<BaziLiuRiOutput[]> {
  return mcpTool<{ items: BaziLiuRiOutput[] }>(ENGINE_TOOLS.BaziLiuRi, {
    standard_time: standardTime,
    longitude,
    count,
    chart,
  }).then(result => result.items)
}

// ── xuankong.chart（玄空飞星宅盘）──
