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
