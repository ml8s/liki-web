// 免费排盘预览：前端 TS 直连引擎 RPC（只读消费，不落库）。
// 引擎 RPC 契约归 liki/engine，方法名/入参以引擎现状为准，绝不修改。

let rpcId = 0

interface RpcEnvelope<T> {
  result?: T
  error?: { code: number; message: string }
}

interface RpcResult<T> {
  data: T
}

export async function engineRpc<T = unknown>(method: string, params: unknown): Promise<T> {
  const response = await fetch('/jsonrpc', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ jsonrpc: '2.0', method, params, id: ++rpcId }),
  })
  if (!response.ok) {
    throw new Error(`引擎服务不可用（HTTP ${response.status}）`)
  }
  const envelope = await response.json() as RpcEnvelope<RpcResult<T>>
  if (envelope.error) {
    throw new Error(`排盘失败：${envelope.error.message}`)
  }
  if (!envelope.result || !('data' in envelope.result)) {
    throw new Error('排盘响应缺少 data')
  }
  return envelope.result.data
}

export interface BaziZhu {
  gan: string
  zhi: string
  na_yin?: string
}

export interface BaziDaYunStep {
  gan: string
  zhi: string
  name: string
  wuxing?: string
  shi_shen?: string
  start_year?: number
  end_year?: number
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
  da_yun?: BaziDaYun
}

export function engineSolarTime(time: string, longitude: number): Promise<{ solar: string; lunar: unknown }> {
  return engineRpc<{ solar: string; lunar: unknown }>('tianwen.time', { time, longitude })
}

export function engineCityCoords(city: string): Promise<{ longitude: number; latitude: number }> {
  return engineRpc<{ longitude: number; latitude: number }>('city.coords', { city })
}

export function engineBaziChart(solarTime: string, gender: string): Promise<BaziChartData> {
  return engineRpc<BaziChartData>('bazi.chart', { solar_time: solarTime, gender })
}

export interface BaziYongShen {
  fu_yi?: { qiangruo?: string; yong?: string; xi?: string; ji?: string }
  tiao_hou?: { yong?: string; xi?: string; season?: string }
  ge_ju?: { ge_ju?: string; yong_fa?: string; yong?: string; xi?: string; ji?: string }
}

export interface BaziShenSha {
  name: string
  category?: string
  description?: string
}

export interface BaziZhuFull extends BaziZhu {
  /** 十神数组（首项为天干十神） */
  shi_shens?: Array<{ shi_shen: string; name?: string }>
  cang_gan?: { main?: string; mid?: string; minor?: string }
  shen_sha?: BaziShenSha[]
  is_void?: boolean
}

export interface BaziSanYuan {
  tai_yuan?: { gan: string; zhi: string }
  ming_gong?: { gan: string; zhi: string }
  shen_gong?: { gan: string; zhi: string }
}

export function engineBaziFullChart(chart: unknown): Promise<{ nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull; yong_shen?: BaziYongShen; san_yuan?: BaziSanYuan }> {
  return engineRpc<{ nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull; yong_shen?: BaziYongShen; san_yuan?: BaziSanYuan }>('bazi.fullchart', { chart })
}

export interface ZiweiGong {
  name: string
  xing_yao: Array<{ name: string; is_zhu_xing?: boolean; liang_du?: string }>
  is_shen_gong?: boolean
}

export interface ZiweiChart {
  gong_wei: ZiweiGong[]
  ming_gong?: number
  shen_gong?: number
}

export function engineZiweiChart(lunar: unknown, gender: string): Promise<unknown> {
  return engineRpc<unknown>('ziwei.chart', { lunar, gender })
}

export function engineZiweiFullChart(chart: unknown): Promise<ZiweiChart> {
  return engineRpc<ZiweiChart>('ziwei.fullchart', { chart })
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
  return engineRpc<BazhaiChart>('bazhai.chart', { birth_year: birthYear, gender })
}

export interface BaziBondPair {
  jia_zhu: string
  yi_zhu: string
  jia_gan: string
  yi_gan: string
  jia_zhi: string
  yi_zhi: string
  gan_guan_xi: string
  zhi_guan_xi: string
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
  return engineRpc<BaziBondResult>('bazi.bond', { a: { chart: chartA }, b: { chart: chartB } })
}