// 共享：公历出生输入 → canonical PaipanContext → 八字/紫微编排。
// 农历只存在于 engine 内部领域与输出投影；前端不再做第二套农历历法。
import {
  type BaziChartData,
  type BaziFullChartData,
  type BazhaiChart,
  type ZiweiChart,
  type ZiweiDaXianItem,
  engineBazhaiChart,
  engineBaziChart,
  engineBaziFullChart,
  engineCityCandidates,
  engineZiweiChart,
  engineZiweiDaXian,
  engineZiweiFullChart,
} from '../api/engine'
import { ApiError } from '../api/http'
import { wallTimeToInstant } from './temporal'

export interface BirthInfo {
  gender: 'female' | 'male'
  /** Gregorian date, YYYY-MM-DD. */
  birth_date: string
  /** Exact civil wall time, HH:mm. */
  birth_time: string
  birth_place?: string
  birth_longitude?: number | null
  birth_timezone?: string
}

/** Engine-boundary time fact. Instant and IANA zone are inseparable. */
export interface PaipanContext {
  instant: string
  timezone: string
  longitude: number
}

/** Display-only zoned formatting; no timezone arithmetic in callers. */
export function formatInstant(
  instant: string,
  timezone: string,
  locale?: string,
  options?: Intl.DateTimeFormatOptions,
): string {
  return new Intl.DateTimeFormat(locale, {
    timeZone: timezone,
    dateStyle: 'medium',
    timeStyle: 'short',
    ...options,
  }).format(new Date(instant))
}

function invalid(code: string, message: string): ApiError {
  return new ApiError(0, code, message)
}

function finiteLongitude(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= -180 && value <= 180
}

function validTimezone(value: unknown): value is string {
  return typeof value === 'string'
    && /^[A-Za-z][A-Za-z0-9_+-]*(\/[A-Za-z0-9_+-]+)+$|^UTC$/.test(value)
}

/** birth info → location → Gregorian wall time → PaipanContext. */
export async function resolvePaipanContext(birth: BirthInfo): Promise<PaipanContext> {
  if (!birth.birth_date || !birth.birth_time) {
    throw invalid('birth_time_required', 'An exact birth time is required')
  }

  let longitude: number
  let timezone: string
  if (birth.birth_longitude !== undefined && birth.birth_longitude !== null) {
    if (!finiteLongitude(birth.birth_longitude)) {
      throw invalid('birth_longitude_invalid', 'Birth longitude must be in [-180, 180]')
    }
    longitude = birth.birth_longitude
    if (!validTimezone(birth.birth_timezone)) {
      throw invalid('birth_timezone_required', 'Birth timezone is required')
    }
    timezone = birth.birth_timezone
  } else {
    const place = birth.birth_place?.trim()
    if (!place) throw invalid('birth_place_required', 'Birth place is required')
    let candidates
    try {
      candidates = (await engineCityCandidates(place)).candidates
    } catch {
      throw invalid('city_not_found', 'Unknown birth place')
    }
    if (candidates.length !== 1) {
      const code = candidates.length ? 'city_ambiguous' : 'city_not_found'
      throw invalid(code, candidates.length ? 'Birth place is ambiguous' : 'Unknown birth place')
    }
    const candidate = candidates[0]
    if (!finiteLongitude(candidate.longitude) || !validTimezone(candidate.timezone)) {
      throw invalid('birth_location_invalid', 'Birth place has no usable longitude and timezone')
    }
    longitude = candidate.longitude
    timezone = candidate.timezone
  }

  try {
    const instant = await wallTimeToInstant(birth.birth_date, birth.birth_time.slice(0, 5), timezone)
    return { instant, timezone, longitude }
  } catch (error) {
    if (error instanceof RangeError && error.message.includes('ambiguous')) {
      throw invalid('birth_time_ambiguous', 'Birth time is ambiguous in this timezone')
    }
    throw invalid('birth_date_invalid', 'Birth date or time is invalid in this timezone')
  }
}

/** birth info → bazi.chart minimal chart. */
export async function baziChartFor(birth: BirthInfo): Promise<BaziChartData> {
  const ctx = await resolvePaipanContext(birth)
  return engineBaziChart({ ...ctx, gender: birth.gender })
}

export interface BaziPan {
  chart: BaziChartData
  full: BaziFullChartData
  ctx: PaipanContext
}

/** 出生信息 → 八字盘（chart + full）。 */
export async function baziPanFor(birth: BirthInfo): Promise<BaziPan> {
  const ctx = await resolvePaipanContext(birth)
  const chart = await engineBaziChart({ ...ctx, gender: birth.gender })
  const full = await engineBaziFullChart(chart)
  return { chart, full, ctx }
}

export interface ZiweiPan {
  chart: ZiweiChart
  daXian: ZiweiDaXianItem[]
}

/** 排盘上下文 → 紫微盘（本命 + 大运）。 */
export async function ziweiPanFor(ctx: PaipanContext, gender: string): Promise<ZiweiPan> {
  const base = await engineZiweiChart({ ...ctx, gender })
  const chart = await engineZiweiFullChart(base)
  const daXian = await engineZiweiDaXian(chart)
  return { chart, daXian }
}

/** 出生年 + 性别 → 八宅盘。 */
export async function bazhaiPanFor(birthYear: number, gender: string): Promise<BazhaiChart> {
  return engineBazhaiChart(birthYear, gender)
}
