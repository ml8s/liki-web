import { describe, expect, it } from 'vitest'
import {
  birthYearFromDate,
  chartInputFromProfileBirth,
  type ProfileBirthSource,
} from '@shared/domain/birth'

const profile = {
  birth: {
    gender: 'female' as const,
    instant: '1990-06-14T23:30:00Z',
    date: '1990-06-15',
    time: '08:30',
    location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
  },
} satisfies ProfileBirthSource

describe('birthYearFromDate', () => {
  it('parses in-range years', () => {
    expect(birthYearFromDate('1990-06-15')).toBe(1990)
    expect(birthYearFromDate('1900-01-01')).toBe(1900)
    expect(birthYearFromDate('2100-12-31')).toBe(2100)
  })

  it('rejects out-of-range and malformed dates', () => {
    expect(birthYearFromDate('1899-12-31')).toBeNull()
    expect(birthYearFromDate('2101-01-01')).toBeNull()
    expect(birthYearFromDate('')).toBeNull()
    expect(birthYearFromDate('not-a-date')).toBeNull()
  })
})

describe('chartInputFromProfileBirth', () => {
  it('projects complete profile facts onto the engine chart input', () => {
    const input = chartInputFromProfileBirth(profile)
    expect(input).toEqual({
      standard_time: '1990-06-15T08:30:00+08:00',
      longitude: 121.4726,
      gender: 'female',
    })
  })

  it('throws on any incomplete fact', () => {
    for (const broken of [
      { ...profile, birth: { ...profile.birth, date: null, time: null, instant: null } },
      { ...profile, birth: { ...profile.birth, location: { longitude: null, timezone: null } } },
      { ...profile, birth: { ...profile.birth, location: { longitude: Number.NaN, timezone: 'UTC' } } },
    ] satisfies ProfileBirthSource[]) {
      expect(() => chartInputFromProfileBirth(broken)).toThrow()
    }
  })
})
