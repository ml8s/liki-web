export type DashboardState =
  | 'no-profile'
  | 'free-profile'
  | 'paid-no-session'
  | 'paid-continue'

export interface DashboardStateInput {
  hasBirthProfile: boolean
  hasEntitlement: boolean
  hasSession: boolean
}

export function resolveDashboardState(input: DashboardStateInput): DashboardState {
  if (!input.hasBirthProfile) return 'no-profile'
  if (!input.hasEntitlement) return 'free-profile'
  return input.hasSession ? 'paid-continue' : 'paid-no-session'
}
