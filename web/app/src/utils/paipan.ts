// 共享：出生信息（档案或手填）→ 真太阳时校正 + bazi.chart。
// 排盘上下文（solar/lunar/longitude）供紫微/八宅等下游消费。
import {
  type BaziChartData,
  engineBaziChart,
  engineCityCoords,
  engineSolarTime,
} from '@/api/engine'
import { ApiError } from '@/api/http'

export interface BirthInfo {
  gender: string
  calendar?: string
  birth_date: string
  birth_time: string
  birth_time_known?: string
  birth_place?: string
}

/** 排盘中间结果：真太阳时校正后的 solar 时间 + 农历 + 经度。 */
export interface PaipanContext {
  /** 校正后（或原始）的 ISO 时间，供 bazi.chart 使用。 */
  solar: string
  /** 农历日期字符串，供 ziwei.chart 使用。 */
  lunar: string
  longitude: number
  /** 是否做了真太阳时校正。 */
  corrected: boolean
}

const ERR_CITY_NOT_FOUND =
  () => new ApiError(0, 'city_not_found', 'Unknown birth place')
const ERR_CITY_INVALID =
  () => new ApiError(0, 'city_invalid', 'Birth place has no coordinates')

/** birth info → coords → 真太阳时校正 → 排盘上下文。 */
export async function resolvePaipanContext(birth: BirthInfo): Promise<PaipanContext> {
  const datetime = `${birth.birth_date}T${birth.birth_time}:00+08:00`
  const known = birth.birth_time_known ?? 'known'
  const useCorrection = known === 'known' && !!birth.birth_place

  let longitude = 120
  if (useCorrection && birth.birth_place) {
    let coords: { longitude?: number } | null
    try {
      coords = await engineCityCoords(birth.birth_place)
    } catch {
      throw ERR_CITY_NOT_FOUND()
    }
    if (!coords || typeof coords.longitude !== 'number') {
      throw ERR_CITY_INVALID()
    }
    longitude = coords.longitude
  }

  const t = await engineSolarTime(datetime, longitude)
  return {
    solar: useCorrection ? t.solar : datetime,
    lunar: t.lunar ?? '',
    longitude,
    corrected: useCorrection,
  }
}

/** birth info → bazi.chart 最小命盘（便捷方法）。 */
export async function baziChartFor(birth: BirthInfo): Promise<BaziChartData> {
  if (birth.calendar === 'lunar') throw new ApiError(0, 'lunar_preview_unsupported', 'Lunar birth input is unsupported')
  const known = birth.birth_time_known ?? 'known'
  if (known === 'unknown' || !birth.birth_time) throw new ApiError(0, 'birth_time_required', 'Birth time is required')
  const ctx = await resolvePaipanContext(birth)
  return engineBaziChart(ctx.solar, birth.gender)
}
