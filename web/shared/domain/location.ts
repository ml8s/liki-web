import { engineCityCandidates } from '../api/engine/city'

export type LocationSource = 'city' | 'browser' | 'manual' | 'saved'

/**
 * Atomic birth/casting location. Longitude and IANA timezone form one wire fact;
 * place is display-only and must never be used to derive a partial location.
 */
export interface BirthLocation {
  place: string
  longitude: number
  timezone: string
  source: LocationSource
  longitude_range?: { west: number; east: number }
}

export function isValidBirthLocation(value: unknown): value is BirthLocation {
  if (typeof value !== 'object' || value === null) return false
  const location = value as Partial<BirthLocation>
  return typeof location.place === 'string'
    && typeof location.longitude === 'number'
    && Number.isFinite(location.longitude)
    && location.longitude >= -180
    && location.longitude <= 180
    && typeof location.timezone === 'string'
    && /^[A-Za-z][A-Za-z0-9_+-]*(\/[A-Za-z0-9_+-]+)+$|^UTC$/.test(location.timezone)
}

/** Free-text location search fallback; callers still choose one atomic candidate. */
export function locationCandidatesFor(place: string) {
  return engineCityCandidates(place)
}
