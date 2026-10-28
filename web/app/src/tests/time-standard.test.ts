import { describe, expect, it } from 'vitest'
import {
  standardTimeForDate,
  standardTimeFromInstant,
  wallTimeToStandardTime,
  yearToStandardTime,
} from '@shared/domain/time'

describe('Engine standard_time projection', () => {
  it('projects a UTC instant into the zone standard offset, not DST', () => {
    expect(standardTimeFromInstant('2024-07-01T04:00:00Z', 'America/New_York'))
      .toBe('2024-06-30T23:00:00-05:00')
    expect(standardTimeFromInstant('2024-01-01T05:00:00Z', 'America/New_York'))
      .toBe('2024-01-01T00:00:00-05:00')
  })

  it('projects civil wall time with the zone standard offset', () => {
    expect(wallTimeToStandardTime('1958-05-12', '06:00', 'Asia/Shanghai'))
      .toBe('1958-05-12T06:00:00+08:00')
  })

  it('projects date-only and year selectors to a stable mid-year representative', () => {
    expect(standardTimeForDate('2026-10-01')).toBe('2026-10-01T12:00:00+08:00')
    expect(yearToStandardTime(2026)).toBe('2026-06-15T12:00:00+08:00')
  })
})
