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

export interface BaziZhuFull extends BaziZhu {
  shi_shen?: string
  cang_gan?: string[]
}

export function engineBaziFullChart(chart: unknown): Promise<{ nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull; yong_shen?: BaziYongShen }> {
  return engineRpc<{ nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull; yong_shen?: BaziYongShen }>('bazi.fullchart', { chart })
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

export interface BazhaiChart {
  ming_gua: { gua: { name: string; wuxing?: string }; group: string }
  ba_zhai_dirs: Record<string, string[]>
}

export function engineBazhaiChart(birthYear: number, gender: string): Promise<BazhaiChart> {
  return engineRpc<BazhaiChart>('bazhai.chart', { birth_year: birthYear, gender })
}