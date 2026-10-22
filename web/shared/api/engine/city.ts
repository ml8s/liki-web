import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

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
  return mcpTool<{ candidates?: CityCandidate[] }>(
    ENGINE_TOOLS.CityCoords,
    { city },
  ).then((result) => ({
    candidates: result.candidates ?? [],
  }))
}
