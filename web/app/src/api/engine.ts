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

export interface BaziChartData {
  nian: BaziZhu
  yue: BaziZhu
  ri: BaziZhu
  shi: BaziZhu
  birth_year?: number
  gender?: string
  zi_shi_rule?: string
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