import {
  reportAppFlowEvent,
  type AppFlowEventPayload,
  type AppFlowFailure,
  type AppFlowIntent,
  type AppFlowSource,
} from '@shared/utils/app-flow-telemetry'
import type { IntentGroupId, IntentBridgeResult } from '@shared/domain/intent-flow'

type ConsultationEvent = Omit<AppFlowEventPayload, 'intentGroup'> & { intentGroup: IntentGroupId }

function report(event: ConsultationEvent): void {
  reportAppFlowEvent(event)
}

/**
 * Keep metric emission out of the consultation view. Values are intentionally
 * tiny enums: this boundary must never receive person IDs or consultation text.
 */
export function useConsultationTelemetry() {
  function trackEntry(input: {
    groupId: IntentGroupId
    seed: AppFlowIntent | null
    source: Extract<AppFlowSource, 'direct' | 'legacy_redirect' | 'marketing' | 'payment'>
    restorable: boolean
  }) {
    report({
      event: 'consultation_entry',
      intentGroup: input.groupId,
      source: input.source,
      intent: input.seed ?? undefined,
      status: input.restorable ? 'restorable' : 'empty',
    })
  }

  function trackIntentGroupSelect(groupId: IntentGroupId) {
    report({ event: 'consultation_intent_group_select', intentGroup: groupId })
  }

  function trackPersonSelect(groupId: IntentGroupId) {
    report({ event: 'consultation_person_select', intentGroup: groupId })
  }

  function trackRestore(input: {
    groupId: IntentGroupId
    source: Extract<AppFlowSource, 'latest' | 'history' | 'payment'>
    restored: boolean
  }) {
    report({
      event: 'consultation_restore',
      intentGroup: input.groupId,
      source: input.source,
      status: input.restored ? 'restored' : 'fresh',
    })
  }

  function trackSubmit(input: {
    groupId: IntentGroupId
    source: Extract<AppFlowSource, 'composer' | 'focused_form' | 'casting'>
    created: boolean
  }) {
    report({
      event: 'consultation_submit',
      intentGroup: input.groupId,
      source: input.source,
      status: input.created ? 'new' : 'existing',
    })
  }

  function trackBridge(input: {
    groupId: IntentGroupId
    seed: AppFlowIntent | null
    result: IntentBridgeResult
  }) {
    const intent = 'intent' in input.result
      ? input.result.intent
      : 'pendingIntent' in input.result
        ? input.result.pendingIntent
        : input.seed
    const status = input.result.status === 'pending-backend'
      ? 'pending_backend'
      : input.result.status === 'cross-group'
        ? 'cross_group'
        : input.result.status
    report({
      event: 'consultation_bridge',
      intentGroup: input.groupId,
      intent: intent ?? undefined,
      status,
    })
  }

  function trackFailure(groupId: IntentGroupId, failure: AppFlowFailure) {
    report({ event: 'consultation_failure', intentGroup: groupId, failure })
  }

  function trackRecovery(groupId: IntentGroupId) {
    report({
      event: 'consultation_recovery',
      intentGroup: groupId,
      source: 'workspace_retry',
    })
  }

  return {
    trackBridge,
    trackIntentGroupSelect,
    trackEntry,
    trackFailure,
    trackPersonSelect,
    trackRecovery,
    trackRestore,
    trackSubmit,
  }
}
