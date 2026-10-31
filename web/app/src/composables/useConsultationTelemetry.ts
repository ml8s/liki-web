import {
  reportAppFlowEvent,
  type AppFlowEventPayload,
  type AppFlowFailure,
  type AppFlowIntent,
  type AppFlowSource,
} from '@shared/utils/app-flow-telemetry'
import type { AppDomainId, LegacyIntentBridgeResult } from '@shared/domain/flow'

type ConsultationEvent = Omit<AppFlowEventPayload, 'domain'> & { domain: AppDomainId }

function report(event: ConsultationEvent): void {
  reportAppFlowEvent(event)
}

/**
 * Keep metric emission out of the consultation view. Values are intentionally
 * tiny enums: this boundary must never receive person IDs or consultation text.
 */
export function useConsultationTelemetry() {
  function trackEntry(input: {
    domainId: AppDomainId
    seed: AppFlowIntent | null
    source: Extract<AppFlowSource, 'direct' | 'legacy_redirect' | 'marketing' | 'payment'>
    restorable: boolean
  }) {
    report({
      event: 'consultation_entry',
      domain: input.domainId,
      source: input.source,
      intent: input.seed ?? undefined,
      status: input.restorable ? 'restorable' : 'empty',
    })
  }

  function trackDomainSelect(domainId: AppDomainId) {
    report({ event: 'consultation_domain_select', domain: domainId })
  }

  function trackPersonSelect(domainId: AppDomainId) {
    report({ event: 'consultation_person_select', domain: domainId })
  }

  function trackRestore(input: {
    domainId: AppDomainId
    source: Extract<AppFlowSource, 'latest' | 'history' | 'payment'>
    restored: boolean
  }) {
    report({
      event: 'consultation_restore',
      domain: input.domainId,
      source: input.source,
      status: input.restored ? 'restored' : 'fresh',
    })
  }

  function trackSubmit(input: {
    domainId: AppDomainId
    source: Extract<AppFlowSource, 'composer' | 'focused_form' | 'casting'>
    created: boolean
  }) {
    report({
      event: 'consultation_submit',
      domain: input.domainId,
      source: input.source,
      status: input.created ? 'new' : 'existing',
    })
  }

  function trackBridge(input: {
    domainId: AppDomainId
    seed: AppFlowIntent | null
    result: LegacyIntentBridgeResult
  }) {
    const intent = 'skill' in input.result
      ? input.result.skill
      : 'legacySkill' in input.result
        ? input.result.legacySkill
        : input.seed
    const status = input.result.status === 'pending-backend'
      ? 'pending_backend'
      : input.result.status === 'cross-domain'
        ? 'cross_domain'
        : input.result.status
    report({
      event: 'consultation_bridge',
      domain: input.domainId,
      intent: intent ?? undefined,
      status,
    })
  }

  function trackFailure(domainId: AppDomainId, failure: AppFlowFailure) {
    report({ event: 'consultation_failure', domain: domainId, failure })
  }

  function trackRecovery(domainId: AppDomainId) {
    report({
      event: 'consultation_recovery',
      domain: domainId,
      source: 'workspace_retry',
    })
  }

  return {
    trackBridge,
    trackDomainSelect,
    trackEntry,
    trackFailure,
    trackPersonSelect,
    trackRecovery,
    trackRestore,
    trackSubmit,
  }
}
