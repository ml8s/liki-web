// 排盘/断语预览：浏览器直连 MCP（/engine/*、/counsel/*，与外部 skill 同一协议栈）。
// 只读、不落库；MCP 工具契约归 liki/engine 与 liki/counsel。
import { ApiError } from './http'

/** 免费 Web UI 可用的 engine MCP 工具名（API 层和 E2E mock 共用）。 */
export const ENGINE_TOOLS = {
  CityCoords: 'city_coords',
  BaziChart: 'bazi_chart',
  BaziFullChart: 'bazi_fullchart',
  BaziLiuNian: 'bazi_liunian',
  ZiweiChart: 'ziwei_chart',
  ZiweiFullChart: 'ziwei_fullchart',
  ZiweiDaXian: 'ziwei_daxian',
  ZiweiLiuNian: 'ziwei_liunian',
  BazhaiChart: 'bazhai_chart',
  BazhaiLayout: 'bazhai_layout',
  XuankongChart: 'xuankong_chart',
  BaziBond: 'bazi_bond',
  LiuyaoQigua: 'liuyao_qigua',
  LiuyaoChart: 'liuyao_chart',
  HuangliDays: 'huangli_days',
} as const

// 直连 MCP（与外部 skill 同一路径与协议）：网关剥离前缀，/engine/* → engine-mcp，
// /counsel/* → counsel-mcp。两服务均为 stateless + JSONResponse，无需 initialize 握手。
interface EngineToolEnvelope {
  result?: { content?: Array<{ type?: string; text?: string }>; isError?: boolean }
  error?: { code?: number; message?: string }
}

/** counsel（裁决/问卦/起名）工具走 /counsel/mcp；排盘/历法工具走 /engine/mcp。 */
export const COUNSEL_TOOLS = new Set<string>([
  'bazi_yongshen', 'compute_factors', 'natal_query', 'period_query',
  'liuyao_query', 'liuyao_snapshot', 'qimen_query', 'qimen_snapshot',
  'naming_surname', 'naming_pick', 'naming_char', 'naming_compose', 'naming_check',
])

function mcpEndpoint(tool: string): string {
  return COUNSEL_TOOLS.has(tool) ? '/counsel/mcp' : '/engine/mcp'
}

// MCP 协议版本：引擎/counsel 强制校验 params._meta 中的 protocolVersion。
const MCP_PROTOCOL_VERSION = '2026-07-28'

export async function engineTool<T = unknown>(tool: string, params: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch(mcpEndpoint(tool), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // MCP Streamable HTTP 规范要求客户端声明接受的两种响应形态。
        Accept: 'application/json, text/event-stream',
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/call',
        params: {
          name: tool,
          arguments: params,
          _meta: {
            'io.modelcontextprotocol/protocolVersion': MCP_PROTOCOL_VERSION,
            'io.modelcontextprotocol/clientCapabilities': {},
          },
        },
      }),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Network request failed')
  }
  if (!response.ok) {
    throw new ApiError(response.status, 'engine_unavailable', `Engine HTTP ${response.status}`)
  }
  let envelope: EngineToolEnvelope
  try {
    envelope = await response.json() as EngineToolEnvelope
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  if (envelope.error) {
    throw new ApiError(response.status, 'engine_tool_failed', envelope.error.message || 'Engine tool failed')
  }
  const content = envelope.result?.content?.[0]
  const text = content?.type === 'text' ? content.text : undefined
  if (envelope.result?.isError || text === undefined) {
    // 工具业务错误：HTTP 200 + isError，错误信息在 content text 内。
    throw new ApiError(response.status, 'engine_tool_failed', text || 'Engine tool failed')
  }
  try {
    return JSON.parse(text) as T
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
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

/** 内置城市表候选（含同名区县、IANA 时区、行政 bbox）；不调用外部地理编码。 */
export interface CityCandidate {
  name: string
  region: string
  city?: string
  country: string
  level: string
  longitude: number
  latitude: number
  timezone?: string
  bbox?: { west: number; east: number; south: number; north: number }
}

export function engineCityCandidates(city: string): Promise<{ candidates: CityCandidate[] }> {
  return engineTool<{ candidates?: CityCandidate[] }>(
    ENGINE_TOOLS.CityCoords,
    { city },
  ).then((result) => ({
    candidates: result.candidates ?? [],
  }))
}

export interface ChartTimeInput {
  instant: string
  timezone: string
  longitude: number
}

export function engineBaziChart(input: ChartTimeInput & { gender: string }): Promise<BaziChartData> {
  return engineTool<BaziChartData>(ENGINE_TOOLS.BaziChart, {
    instant: input.instant,
    timezone: input.timezone,
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

export function engineZiweiChart(input: ChartTimeInput & { gender: string }): Promise<ZiweiChart> {
  return engineTool<ZiweiChart>(ENGINE_TOOLS.ZiweiChart, {
    instant: input.instant,
    timezone: input.timezone,
    longitude: input.longitude,
    gender: input.gender,
  })
}

export function engineZiweiFullChart(chart: unknown): Promise<ZiweiChart> {
  return engineTool<ZiweiChart>(ENGINE_TOOLS.ZiweiFullChart, { chart })
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
  return engineTool<ZiweiDaXianItem[]>(ENGINE_TOOLS.ZiweiDaXian, { chart })
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
  return engineTool<ZiweiLiuNianData>(ENGINE_TOOLS.ZiweiLiuNian, { chart, lunar_year: lunarYear })
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
  return engineTool<{ casting: LiuyaoCasting }>(ENGINE_TOOLS.LiuyaoQigua, input)
}

/** 六爻装卦：传入 casting + 真太阳时，返回卦象/用神/应期。 */
export function engineLiuyaoChart(
  input: ChartTimeInput & { casting: LiuyaoCasting; yongShen?: string },
): Promise<LiuyaoChartResult> {
  const params: Record<string, unknown> = {
    instant: input.instant,
    timezone: input.timezone,
    longitude: input.longitude,
    casting: input.casting,
  }
  if (input.yongShen) params.yong_shen = input.yongShen
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
/** bazi_yongshen（counsel）裁决结果：用神/喜忌 + 来源与置信度。 */
export interface BaziYongShenResult {
  selected: {
    yong: string
    xi: string[]
    ji: string[]
    /** 判据来源：tiao_hou / fu_yi / ge_ju … */
    source: string
    confidence: string
  }
  status?: string
  conflicts?: Array<{ code?: string; message?: string }>
  rule_id?: string
  ge_ju_context?: Record<string, unknown>
}

/** 起名/择吉的用神裁决（counsel 域，直连 /counsel/mcp）。 */
export function engineBaziYongShen(chart: unknown): Promise<BaziYongShenResult> {
  return engineTool<BaziYongShenResult>('bazi_yongshen', { chart })
}

export function engineHuangliDays(event: string, startDate: string, count: number): Promise<HuangliDay[]> {
  return engineTool<HuangliDay[]>(ENGINE_TOOLS.HuangliDays, { event, start_date: startDate, count })
}

// ── bazi.liunian（流年盘）──

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

/** 流年盘（bazi.liunian）：最小命盘 + 目标年份。 */
export function engineBaziLiunian(chart: unknown, year: number): Promise<BaziLiuNianOutput> {
  return engineTool<BaziLiuNianOutput>(ENGINE_TOOLS.BaziLiuNian, { chart, year })
}

// ── xuankong.chart（玄空飞星宅盘）──

export interface XuankongStar {
  number: number
  color?: string
  name: string
  wuxing?: string
}

export interface XuankongPalace {
  /** 洛书宫数 1-9。 */
  gong_num: number
  yun_xing: XuankongStar
  shan_xing: XuankongStar
  xiang_xing: XuankongStar
}

export interface XuankongYun {
  year: number
  yuan?: string
  yun_number: number
  yun_name: string
  start_year: number
  end_year: number
}

export interface XuankongFourSituation {
  name: string
  sit_mountain_timely: boolean
  face_facing_timely: boolean
  sit_facing_timely: boolean
  face_mountain_timely: boolean
}

export interface XuankongShouShanChuSha {
  zheng_shen: number
  ling_shen: number
  shou_shan: boolean
  chu_sha: boolean
  assessment?: string
}

export interface XuankongXingJiaHui {
  shan_num: number
  xiang_num: number
  name: string
  meaning: string
}

export interface XuankongChart {
  yun: XuankongYun
  /** 九宫（洛书宫数 1-9，非渲染顺序）。 */
  gong_wei: XuankongPalace[]
  four_situation: XuankongFourSituation
  fu_yin_layers?: string[]
  fan_yin_layers?: string[]
  xing_jia_hui?: XuankongXingJiaHui[]
  zuo_shan: number
  zuo_shan_name: string
  xiang_shan: number
  xiang_shan_name: string
  shou_shan_chu_sha: XuankongShouShanChuSha
}

/** 玄空宅盘：period_date=立极日期（YYYY-MM-DD），坐山/向山为二十四山 index（0=子…23=壬，向=坐+12）。 */
export function engineXuankongChart(periodDate: string, zuoShan: number, xiangShan: number): Promise<XuankongChart> {
  return engineTool<XuankongChart>(ENGINE_TOOLS.XuankongChart, {
    period_date: periodDate,
    zuo_shan: zuoShan,
    xiang_shan: xiangShan,
  })
}

// ── bazhai.layout（八宅门主灶布局判断）──

export interface BazhaiDoorStove {
  direction: string
  gua_name: string
  wuxing?: string
  /** 游年九星（伏位/生气/延年/天医/祸害/六煞/五鬼/绝命）。 */
  youxing: string
  rating: string
  group?: string
  /** 与命卦同组/异组。 */
  match: string
}

export interface BazhaiLayout {
  group: string
  ming_gua: string
  door: BazhaiDoorStove
  master: BazhaiDoorStove
  stove: BazhaiDoorStove
}

/** 八宅布局：命卦 + 门/主/灶三卦（乾坎艮震巽离坤兑）。 */
export function engineBazhaiLayout(mingGua: string, doorGua: string, masterGua: string, stoveGua: string): Promise<BazhaiLayout> {
  return engineTool<BazhaiLayout>(ENGINE_TOOLS.BazhaiLayout, {
    ming_gua: mingGua,
    door_gua: doorGua,
    master_gua: masterGua,
    stove_gua: stoveGua,
  })
}
