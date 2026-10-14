import { describe, expect, it } from 'vitest'
import { firstQueryValue, queryChoice, queryText } from '@/utils/query'

describe('query state', () => {
  it('reads the first array value', () => {
    expect(firstQueryValue(['requested', 'paid'])).toBe('requested')
    expect(firstQueryValue(42)).toBe('')
  })

  it('rejects unknown enum values', () => {
    expect(queryChoice({ status: 'paid' }, 'status', ['requested','paid'], 'requested')).toBe('paid')
    expect(queryChoice({ status: 'admin' }, 'status', ['requested'], 'requested')).toBe('requested')
  })

  it('normalizes query text', () => {
    expect(queryText({ q: ' KOL-123 ' }, 'q')).toBe('KOL-123')
    expect(queryText({}, 'q')).toBe('')
  })
})

import { canonicalQuery } from '@/utils/query'

describe('canonical admin query', () => {
  it('removes defaults, empty values, and first-page state', () => {
    expect(canonicalQuery({
      applicationRole: '',
      applicationStatus: 'pending',
      partnerRole: '',
      partnerStatus: '',
      q: '',
      page: 1,
    }, {
      applicationRole: '',
      applicationStatus: 'pending',
      partnerRole: '',
      partnerStatus: '',
      q: '',
      page: 1,
    })).toEqual({})
  })

  it('keeps only meaningful differences', () => {
    expect(canonicalQuery({
      applicationRole: 'kol',
      applicationStatus: 'pending',
      partnerStatus: 'frozen',
      q: 'KOL-123',
      page: 3,
    }, {
      applicationRole: '',
      applicationStatus: 'pending',
      partnerRole: '',
      partnerStatus: '',
      q: '',
      page: 1,
    })).toEqual({
      applicationRole: 'kol',
      partnerStatus: 'frozen',
      q: 'KOL-123',
      page: 3,
    })
  })
})
