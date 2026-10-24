/** Local city search over the canonical cities_data.json (same file the engine embeds). */
import { CITY_DATA_URL } from '../generated/cityData'

export type CityLocale = 'zh-Hans' | 'zh-Hant' | 'en'

type LocalizedNames = Partial<Record<CityLocale, string>>

export interface CityRecord {
  id?: string
  name: string
  region: string
  city?: string
  lng: number
  lat: number
  tz?: string
  bbox?: { west: number; east: number; south: number; north: number }
  aliases?: string[]
  names?: LocalizedNames
  region_names?: LocalizedNames
}

export interface CityIndex {
  meta?: { schema_version?: number }
  counties: CityRecord[]
  cities: CityRecord[]
  world: CityRecord[]
}

export interface CityMatch extends CityRecord {
  level: 'county' | 'city' | 'world'
  /** Engine candidate carries only the east/west range relevant to time boundary. */
  longitude_range?: { west: number; east: number }
}

let cache: CityIndex | null = null
let loading: Promise<CityIndex> | null = null

export function resetCityIndexCache(): void {
  cache = null
  loading = null
}

export async function loadCityIndex(url = CITY_DATA_URL): Promise<CityIndex> {
  if (cache) return cache
  if (!loading) {
    loading = fetch(url)
      .then((response) => {
        if (!response.ok) throw new Error(`city index: HTTP ${response.status}`)
        return response.json() as Promise<CityIndex>
      })
      .then((data) => {
        const schema = data.meta?.schema_version
        if (schema !== undefined && schema !== 2) {
          throw new Error(`city index: unsupported schema ${schema}`)
        }
        cache = data
        return data
      })
      .finally(() => { loading = null })
  }
  return loading
}

const SUFFIXES = ['特别行政区', '自治州', '自治县', '自治区', '地区', '盟', '市', '区', '县', '旗']

function stripSuffix(value: string): string {
  for (const suffix of SUFFIXES) {
    if (value.endsWith(suffix)) return value.slice(0, -suffix.length)
  }
  return value
}

function normalize(value: string): string {
  return value.normalize('NFKC').trim().toLocaleLowerCase()
}

function isLatin(value: string): boolean {
  return /^[a-z][a-z .'-]*$/.test(value)
}

function latinWordMatch(name: string, text: string): boolean {
  const escaped = text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return new RegExp(`(^|[^a-z])${escaped}([^a-z]|$)`, 'i').test(name)
}

function matchClass(query: string, record: CityRecord): number | null {
  const text = normalize(query)
  const base = stripSuffix(text)
  const names = [
    record.name,
    ...Object.values(record.names ?? {}),
    ...(record.aliases ?? []),
  ].filter((name): name is string => Boolean(name)).map(normalize)
  if (names.includes(text)) return 0
  if (base && names.some((name) => stripSuffix(name) === base)) return 1
  if (isLatin(text)) {
    if (names.some((name) => latinWordMatch(name, text))) return 2
    if (base && names.some((name) => latinWordMatch(name, base))) return 3
  } else {
    if (names.some((name) => name.startsWith(text))) return 2
    if (base && names.some((name) => name.includes(base))) return 3
  }
  return null
}

export function cachedCityById(id?: string): CityRecord | undefined {
  if (!id || !cache) return undefined
  for (const table of [cache.cities, cache.counties, cache.world]) {
    const record = table.find(item => item.id === id)
    if (record) return record
  }
  return undefined
}

export function searchCities(index: CityIndex, query: string, limit = 20): CityMatch[] {
  const text = query.trim()
  if (!text) return []
  const tables: Array<[CityMatch['level'], CityRecord[]]> = [
    ['city', index.cities],
    ['county', index.counties],
    ['world', index.world],
  ]
  const out: Array<{ match: CityMatch; cls: number; level: number }> = []
  for (const [level, records] of tables) {
    for (const record of records) {
      const cls = matchClass(text, record)
      if (cls === null) continue
      out.push({ match: { ...record, level }, cls, level: level === 'city' ? 0 : level === 'county' ? 1 : 2 })
    }
  }
  out.sort((a, b) => a.cls - b.cls || a.level - b.level || b.match.lng - a.match.lng)
  // An exact/alias hit means prefix-only rows are noise (Changshan for
  // changsha). Keep the exact tier when it exists; partial queries still
  // search prefixes.
  const best = out[0]?.cls ?? 0
  const filtered = best <= 1 ? out.filter((item) => item.cls === best) : out
  return filtered.slice(0, limit).map((item) => item.match)
}
