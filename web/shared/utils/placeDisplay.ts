/**
 * Place display is a presentation concern, not a storage concern.
 *
 * The city index and engine return canonical Chinese names. We keep the
 * canonical record as data and derive a locale-specific label at the edge.
 * This prevents duplicated labels (region + city), leaked Simplified Chinese
 * on Traditional pages, and hard-coded hemisphere markers in copy.
 */
import { cachedCityById } from './citySearch'
import { isCurrentLocation } from '../domain/location'

export type PlaceLocale = 'zh-Hans' | 'zh-Hant' | 'en'

export interface PlaceRecord {
  name: string
  region: string
  city?: string
  aliases?: string[]
  names?: Partial<Record<PlaceLocale, string>>
  region_names?: Partial<Record<PlaceLocale, string>>
}

function englishAlias(record: PlaceRecord): string | undefined {
  return record.aliases?.find(alias => (
    /^[A-Za-z][A-Za-z .'-]+$/.test(alias)
    && alias.length > 3
    && !/^[A-Z]{2,4}$/.test(alias)
  ))
}

export function localizedCityName(record: PlaceRecord, locale: PlaceLocale): string {
  return record.names?.[locale] ?? (locale === 'en' ? englishAlias(record) : undefined) ?? record.name
}

export function localizedRegion(record: PlaceRecord, locale: PlaceLocale): string {
  return record.region_names?.[locale] ?? record.region
}

export function formatSavedPlace(
  location: { place: string; place_id?: string; location_version?: number },
  locale: PlaceLocale,
): string {
  if (!isCurrentLocation(location)) return ''
  if (location.place_id) {
    const record = cachedCityById(location.place_id)
    if (record) return formatPlace(record, locale)
  }
  return location.place
}

/**
 * Compose a display label without repeating the city when the region already
 * names it (e.g. 中国香港 + 香港).
 */
export function formatPlace(record: PlaceRecord, locale: PlaceLocale): string {
  const city = localizedCityName(record, locale)
  const region = localizedRegion(record, locale)
  if (!region || region === city) return city
  if (region.includes(city)) return region
  if (city.includes(region)) return city
  return locale === 'en' ? `${city}, ${region}` : `${region}${city}`
}

export function formatLongitude(value: number, locale: PlaceLocale): string {
  const magnitude = Math.abs(value).toFixed(4)
  if (locale === 'en') return `${magnitude}°${value < 0 ? 'W' : 'E'}`
  if (locale === 'zh-Hant') return `${value < 0 ? '西經' : '東經'} ${magnitude}°`
  return `${value < 0 ? '西经' : '东经'} ${magnitude}°`
}
