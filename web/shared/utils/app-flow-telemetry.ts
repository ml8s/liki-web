import { postTelemetryJSON, telemetryDisabled } from './telemetry'

/**
 * PII-free Consultation funnel telemetry.
 *
 * Person IDs, names, questions, birth data, session IDs, and raw API errors
 * are intentionally absent. Every field is a closed vocabulary shared with the
 * backend so production metrics have stable cardinality.
 */
export type AppFlowEvent =
  | 'consultation_entry'
  | 'consultation_domain_select'
  | 'consultation_person_select'
  | 'consultation_restore'
  | 'consultation_submit'
  | 'consultation_bridge'
  | 'consultation_failure'
  | 'consultation_recovery'

export type AppFlowDomain = 'natal' | 'divination' | 'fengshui' | 'naming'

export type AppFlowSource =
  | 'direct'
  | 'legacy_redirect'
  | 'marketing'
  | 'payment'
  | 'latest'
  | 'history'
  | 'composer'
  | 'focused_form'
  | 'casting'
  | 'workspace_retry'

export type AppFlowScenario =
  | 'natal'
  | 'liunian'
  | 'hepan'
  | 'huangli'
  | 'divination'
  | 'fengshui'
  | 'naming'

export type AppFlowStatus =
  | 'empty'
  | 'restorable'
  | 'restored'
  | 'fresh'
  | 'new'
  | 'existing'
  | 'ok'
  | 'cross_domain'
  | 'pending_backend'
  | 'invalid'

export type AppFlowFailure =
  | 'offline'
  | 'unreachable'
  | 'permission'
  | 'server'
  | 'unknown'

export interface AppFlowEventPayload {
  event: AppFlowEvent
  domain?: AppFlowDomain
  source?: AppFlowSource
  scenario?: AppFlowScenario
  status?: AppFlowStatus
  failure?: AppFlowFailure
}

export function reportAppFlowEvent(payload: AppFlowEventPayload): void {
  if (telemetryDisabled()) return
  postTelemetryJSON('/api/analytics/product-event', payload)
}
