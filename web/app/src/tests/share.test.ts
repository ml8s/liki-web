import { describe, expect, it } from 'vitest'
import { buildReferralLink } from '@/utils/share'

describe('buildReferralLink', () => {
  it('builds a referral entry URL from the partner code', () => {
    expect(buildReferralLink('KOL123', 'https://liki.hk')).toBe('https://liki.hk/r/KOL123')
  })

  it('encodes unsafe path values and returns empty for missing codes', () => {
    expect(buildReferralLink('abc def', 'https://liki.hk')).toBe('https://liki.hk/r/abc%20def')
    expect(buildReferralLink('  ', 'https://liki.hk')).toBe('')
  })
})
