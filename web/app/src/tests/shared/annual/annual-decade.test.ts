import { describe, expect, it } from 'vitest'
import {
  mergeAnnualDecade,
  type BaziLiuNianOutput,
  type ZiweiLiuNianData,
} from '@shared/domain/natal'

const bazi = (year: number): BaziLiuNianOutput => ({
  year,
  nian_name: '丙午',
  shi_shen: '正官',
})

const ziwei = (_year: number): ZiweiLiuNianData => ({
  ming_gong: '午',
  ming_gong_name: '命宫',
  zhi: '午',
})

describe('mergeAnnualDecade', () => {
  it('builds ten consecutive years from the selected start year', () => {
    const years = mergeAnnualDecade(2027)
    expect(years.map(item => item.year)).toEqual(Array.from({ length: 10 }, (_, index) => 2027 + index))
    expect(years.every(item => item.availability === 'missing')).toBe(true)
  })

  it('marks ready, partial, and missing years without inventing data', () => {
    const years = mergeAnnualDecade(
      2027,
      [bazi(2027), bazi(2028), bazi(2030)],
      [ziwei(2027), ziwei(2028)],
    )

    expect(years.find(item => item.year === 2027)?.availability).toBe('ready')
    expect(years.find(item => item.year === 2028)?.availability).toBe('ready')
    expect(years.find(item => item.year === 2029)?.availability).toBe('partial')
    expect(years.find(item => item.year === 2030)?.availability).toBe('missing')
    expect(years.find(item => item.year === 2031)?.availability).toBe('missing')
  })

  it('maps range items by anchor order and requested start year', () => {
    const years = mergeAnnualDecade(2027, [bazi(2030)], [ziwei(2030)])
    expect(years[0]?.year).toBe(2027)
    expect(years[0]?.availability).toBe('ready')
    expect(years.slice(1).every(item => item.availability === 'missing')).toBe(true)
  })
})
