import { describe, expect, it } from 'vitest'
import { ApiError } from '@/api/http'
import { classifyResourceFailure } from '@/utils/resourceStatus'

describe('classifyResourceFailure', () => {
  it('separates unreachable services from permission failures', () => {
    expect(classifyResourceFailure(new ApiError(0, 'network_error', 'offline'))).toBe('unreachable')
    expect(classifyResourceFailure(new ApiError(403, 'forbidden', 'denied'))).toBe('permission')
    expect(classifyResourceFailure(new ApiError(500, 'internal_error', 'boom'))).toBe('server')
    expect(classifyResourceFailure(new TypeError('Failed to fetch'))).toBe('unreachable')
    expect(classifyResourceFailure(new Error('unexpected'))).toBe('unknown')
  })
})
