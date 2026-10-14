import { describe, expect, it } from 'vitest'
import { resolveDashboardState } from '@/composables/dashboardState'

describe('resolveDashboardState', () => {
  it('guides users without a profile to create one', () => {
    expect(resolveDashboardState({
      hasBirthProfile: false,
      hasEntitlement: false,
      hasSession: false,
    })).toBe('no-profile')
  })

  it('guides profile owners without entitlement to paid interpretation', () => {
    expect(resolveDashboardState({
      hasBirthProfile: true,
      hasEntitlement: false,
      hasSession: false,
    })).toBe('free-profile')
  })

  it('guides entitled users without a session to start one', () => {
    expect(resolveDashboardState({
      hasBirthProfile: true,
      hasEntitlement: true,
      hasSession: false,
    })).toBe('paid-no-session')
  })

  it('continues the latest session when one exists', () => {
    expect(resolveDashboardState({
      hasBirthProfile: true,
      hasEntitlement: true,
      hasSession: true,
    })).toBe('paid-continue')
  })
})
