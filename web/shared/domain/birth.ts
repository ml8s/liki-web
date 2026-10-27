import { ApiError } from '../api/http'
import type { BirthLocation } from './location'
import { wallTimeToInstant, type ChartInput } from './time'

export type BirthGender = 'female' | 'male'

/** Exact user-facing birth facts. Incomplete data is not a BirthInput. */
export interface BirthInput {
  gender: BirthGender
  /** Gregorian date projection, YYYY-MM-DD. */
  date: string
  /** Exact civil wall-clock time, HH:mm. */
  time: string
  location: BirthLocation
}

function invalid(code: string, message: string): ApiError {
  return new ApiError(0, code, message)
}

/** Birth wall time + location -> canonical engine chart input. */
export async function chartInputFromBirth(birth: BirthInput): Promise<ChartInput> {
  const { longitude, timezone } = birth.location
  if (!Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
    throw invalid('birth_longitude_invalid', 'Birth longitude must be in [-180, 180]')
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(birth.date) || !/^\d{2}:\d{2}$/.test(birth.time)) {
    throw invalid('birth_time_required', 'An exact birth date and time are required')
  }

  try {
    return {
      instant: await wallTimeToInstant(birth.date, birth.time, timezone),
      timezone,
      longitude,
    }
  } catch (error) {
    if (error instanceof RangeError && error.message.includes('ambiguous')) {
      throw invalid('birth_time_ambiguous', 'Birth time is ambiguous in this timezone')
    }
    throw invalid('birth_date_invalid', 'Birth date or time is invalid in this timezone')
  }
}

/** Birth year parsed from a YYYY-MM-DD date, bounded to engine-supported years. */
export function birthYearFromDate(date: string, min = 1900, max = 2100): number | null {
  const year = Number.parseInt(date.slice(0, 4), 10)
  return Number.isFinite(year) && year >= min && year <= max ? year : null
}

/** Structural source: the saved birth facts a profile contributes to a chart. */
export interface ProfileBirthSource {
  birth: {
    gender: BirthGender
    /** Nullable mirrors of the saved-profile wire shape; guarded below. */
    instant: string | null
    date: string | null
    time: string | null
    location: { longitude: number | null; timezone: string | null }
  }
}

/**
 * Saved profile birth facts -> engine chart input. Throws when the facts are
 * incomplete; callers that prefer soft failures should guard before calling.
 */
export function chartInputFromProfileBirth(
  profile: ProfileBirthSource,
): ChartInput & { gender: BirthGender } {
  const birth = profile.birth
  const longitude = birth.location?.longitude
  const timezone = birth.location?.timezone
  if (!birth.instant || !timezone
    || typeof longitude !== 'number' || !Number.isFinite(longitude)) {
    throw invalid('birth_time_required', 'A complete birth profile is required')
  }
  return {
    instant: birth.instant,
    timezone,
    longitude,
    gender: birth.gender,
  }
}
