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
