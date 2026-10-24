import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'
import { formatLongitude, formatPlace } from '@shared/utils/placeDisplay'

const hongKong = { name: '香港', region: '中国香港', aliases: ['Hong Kong', 'HK', '香港'], names: { 'zh-Hans': '香港', 'zh-Hant': '香港', en: 'Hong Kong' }, region_names: { 'zh-Hans': '中国香港', 'zh-Hant': '中國香港', en: 'Hong Kong' } }
const macau = { name: '澳门', region: '中国澳门', aliases: ['Macau', 'MO', '澳门'], names: { 'zh-Hans': '澳门', 'zh-Hant': '澳門', en: 'Macau' }, region_names: { 'zh-Hans': '中国澳门', 'zh-Hant': '中國澳門', en: 'Macau' } }
const taipei = { name: '台北', region: '中国台湾', aliases: ['Taipei', 'TPE', '台北'], names: { 'zh-Hans': '台北', 'zh-Hant': '台北', en: 'Taipei' }, region_names: { 'zh-Hans': '中国台湾', 'zh-Hant': '中國台灣', en: 'Taiwan' } }
const beijing = { name: '北京市', region: '北京市', names: { 'zh-Hans': '北京市', 'zh-Hant': '北京市', en: 'Beijing' }, region_names: { 'zh-Hans': '北京市', 'zh-Hant': '北京市', en: 'Beijing' } }
const london = { name: '伦敦', region: '英国', aliases: ['London', 'LON', '伦敦'], names: { 'zh-Hans': '伦敦', 'zh-Hant': '倫敦', en: 'London' }, region_names: { 'zh-Hans': '英国', 'zh-Hant': '英國', en: 'United Kingdom' } }
const newYork = { name: '纽约', region: '美国', aliases: ['New York City', 'NYC', '纽约'], names: { 'zh-Hans': '纽约', 'zh-Hant': '紐約', en: 'New York' }, region_names: { 'zh-Hans': '美国', 'zh-Hant': '美國', en: 'United States' } }

describe('formatPlace', () => {
  it('does not repeat a city already contained in the region', () => {
    expect(formatPlace(hongKong, 'zh-Hant')).toBe('中國香港')
    expect(formatPlace(macau, 'zh-Hant')).toBe('中國澳門')
    expect(formatPlace(beijing, 'zh-Hans')).toBe('北京市')
  })

  it('localizes region and city for English', () => {
    expect(formatPlace(hongKong, 'en')).toBe('Hong Kong')
    expect(formatPlace(taipei, 'en')).toBe('Taipei, Taiwan')
    expect(formatPlace(london, 'en')).toBe('London, United Kingdom')
    expect(formatPlace(newYork, 'en')).toBe('New York, United States')
  })

  it('uses Traditional Chinese labels', () => {
    expect(formatPlace(london, 'zh-Hant')).toBe('英國倫敦')
    expect(formatPlace(newYork, 'zh-Hant')).toBe('美國紐約')
    expect(formatPlace(taipei, 'zh-Hant')).toBe('中國台灣台北')
  })
})

describe('formatLongitude', () => {
  it('uses the hemisphere that matches the sign', () => {
    expect(formatLongitude(114.169, 'en')).toBe('114.1690°E')
    expect(formatLongitude(-0.128, 'en')).toBe('0.1280°W')
    expect(formatLongitude(-74.006, 'en')).toBe('74.0060°W')
  })

  it('localizes the direction label', () => {
    expect(formatLongitude(114.169, 'zh-Hant')).toBe('東經 114.1690°')
    expect(formatLongitude(-0.128, 'zh-Hant')).toBe('西經 0.1280°')
    expect(formatLongitude(-74.006, 'zh-Hans')).toBe('西经 74.0060°')
  })
})


describe('city index coverage', () => {
  it('renders every world city with an English label', () => {
    const file = path.resolve(process.cwd(), '../public/data/cities_data.json')
    const index = JSON.parse(fs.readFileSync(file, 'utf8')) as {
      world: Array<{ name: string; region: string; aliases?: string[] }>
    }
    const missing = index.world
      .filter(record => /[\u3400-\u9fff]/.test(formatPlace(record, 'en')))
      .map(record => record.name)
    expect(missing).toEqual([])
  })
})
