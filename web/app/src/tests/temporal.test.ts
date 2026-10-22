import { describe, expect, it } from 'vitest'
import { wallTimeToInstantCandidates, wallTimeToInstant } from '@shared/utils/temporal'

describe('temporal', () => {
  it('converts a Gregorian wall time to UTC', async () => {
    await expect(wallTimeToInstant('1958-05-12', '06:00', 'Asia/Shanghai'))
      .resolves.toBe('1958-05-11T22:00:00Z')
  })

  it('uses China DST rules for historical dates', async () => {
    // 1990-05-12 08:30 Asia/Shanghai was UTC+09:00 under China DST.
    await expect(wallTimeToInstant('1990-05-12', '08:30', 'Asia/Shanghai'))
      .resolves.toBe('1990-05-11T23:30:00Z')
  })

  it('returns no candidate for a DST gap', async () => {
    await expect(wallTimeToInstantCandidates('2024-03-10', '02:30', 'America/New_York'))
      .resolves.toEqual([])
  })

  it('returns both candidates for a DST fold', async () => {
    const candidates = await wallTimeToInstantCandidates('2024-11-03', '01:30', 'America/New_York')
    expect(candidates).toEqual([
      '2024-11-03T05:30:00Z',
      '2024-11-03T06:30:00Z',
    ])
  })

  it('rejects ambiguous wall time unless the caller resolves it', async () => {
    await expect(wallTimeToInstant('2024-11-03', '01:30', 'America/New_York'))
      .rejects.toThrow(RangeError)
  })
})
