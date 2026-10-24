import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'
import { searchCities, type CityIndex, type CityMatch } from '@shared/utils/citySearch'
import { formatPlace } from '@shared/utils/placeDisplay'

const index = JSON.parse(
  fs.readFileSync(path.resolve(process.cwd(), '../public/data/cities_data.json'), 'utf8'),
) as CityIndex

function first(query: string): CityMatch {
  const match = searchCities(index, query, 20)[0]
  if (!match) throw new Error(`no match for ${query}`)
  return match
}

describe('city search localization', () => {
  it('finds the same Chinese city from Simplified, Traditional and English', () => {
    for (const query of ['北京', '北京市', 'Beijing', 'beijing']) {
      expect(first(query).id).toBe('CN-110000')
    }
  })

  it('finds the same overseas city from Chinese and English', () => {
    for (const query of ['臺北', '台北', 'Taipei']) {
      expect(first(query).id).toBe('GN-1668341')
    }
    for (const query of ['Tokyo', 'tokyo', '東京']) {
      expect(first(query).id).toBe('GN-1850147')
    }
  })

  it('finds subdivisions by English or pinyin', () => {
    expect(first('Haidian').id).toBe('CN-110108')
  })

  it('keeps city before county for ambiguous names', () => {
    expect(first('长沙').name).toBe('长沙市')
  })

  it('renders the selected record in the active locale', () => {
    const beijing = first('Beijing')
    expect(formatPlace(beijing, 'zh-Hans')).toBe('北京市')
    expect(formatPlace(beijing, 'zh-Hant')).toBe('北京市')
    expect(formatPlace(beijing, 'en')).toBe('Beijing')
    const haidian = first('Haidian')
    expect(formatPlace(haidian, 'en')).toBe('Haidian, Beijing')
    expect(formatPlace(haidian, 'zh-Hant')).toBe('北京市海淀區')
  })
})
