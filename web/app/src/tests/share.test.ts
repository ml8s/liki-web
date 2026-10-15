import { describe, expect, it } from 'vitest'
import { buildReferralLink } from '@/utils/share'

describe('buildReferralLink', () => {
  it('builds a referral entry URL from the partner code', () => {
    expect(buildReferralLink('KOL123', { origin: 'https://liki.hk' })).toBe('https://liki.hk/r/KOL123')
  })

  it('encodes unsafe path values and returns empty for missing codes', () => {
    expect(buildReferralLink('abc def', { origin: 'https://liki.hk' })).toBe('https://liki.hk/r/abc%20def')
    expect(buildReferralLink('  ', { origin: 'https://liki.hk' })).toBe('')
  })

  it('appends only allowlisted same-origin targets', () => {
    expect(buildReferralLink('KOL123', { target: '/zh-Hant/naming', origin: 'https://liki.hk' }))
      .toBe('https://liki.hk/r/KOL123?to=%2Fzh-Hant%2Fnaming')
    expect(buildReferralLink('KOL123', { target: '/app', origin: 'https://liki.hk' }))
      .toBe('https://liki.hk/r/KOL123?to=%2Fapp')
  })

  it('ignores private or external targets', () => {
    expect(buildReferralLink('KOL123', { target: '/app/account', origin: 'https://liki.hk' }))
      .toBe('https://liki.hk/r/KOL123')
    expect(buildReferralLink('KOL123', { target: 'https://evil.example', origin: 'https://liki.hk' }))
      .toBe('https://liki.hk/r/KOL123')
  })
})
