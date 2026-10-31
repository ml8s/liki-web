import intentGroupContract from '../config/intent-groups.json'
import intentContract from '../config/intents.json'

export type IntentGroupId = 'natal' | 'divination' | 'fengshui' | 'naming'
export type IntentId =
  | 'natal'
  | 'liunian'
  | 'hepan'
  | 'divination'
  | 'huangli'
  | 'fengshui'
  | 'naming'
export type IntentProfilePolicy = 'required' | 'optional' | 'partial'

export interface IntentGroup {
  id: IntentGroupId
  labelKey: string
  baseIntent: IntentId
  acceptedIntents: readonly IntentId[]
  bridgeIntents: readonly IntentId[]
  defaultPath: string
  profilePolicy: IntentProfilePolicy
}

export interface IntentBridgeInput {
  groupId: string
  personId?: string
  seed?: string | null
  slotFormResult?: IntentId | null
}

export type IntentBridgeResult =
  | {
      status: 'ok'
      groupId: IntentGroupId
      intent: IntentId
      fallbackUsed: boolean
      reason: 'seed' | 'slot-form' | 'group-base'
    }
  | {
      status: 'cross-group'
      groupId: IntentGroupId
      suggestedGroupId: IntentGroupId
    }
  | {
      status: 'pending-backend'
      groupId: IntentGroupId
      pendingIntent: IntentId
    }
  | {
      status: 'invalid'
      reason: 'unknown-group' | 'empty-routing-input'
    }

export const intentGroups: readonly IntentGroup[] = intentGroupContract.intentGroups.map(group => ({
  id: group.id as IntentGroupId,
  labelKey: group.labelKey,
  baseIntent: group.baseIntent as IntentId,
  acceptedIntents: group.acceptedIntents as IntentId[],
  bridgeIntents: group.bridgeIntents as IntentId[],
  defaultPath: group.defaultPath,
  profilePolicy: group.profilePolicy as IntentProfilePolicy,
}))

const groupByIntent = new Map<IntentId, IntentGroup>()
for (const group of intentGroups) {
  for (const intent of group.acceptedIntents) groupByIntent.set(intent, group)
}

const groupById = new Map<IntentGroupId, IntentGroup>(
  intentGroups.map(group => [group.id, group]),
)

const intentIds = new Set(
  intentContract.intents.map(intent => intent.id as IntentId),
)

export function normalizeIntentGroup(value: unknown): IntentGroupId | null {
  const id = String(value ?? '')
  return groupById.has(id as IntentGroupId) ? id as IntentGroupId : null
}

export function normalizeIntent(value: unknown): IntentId | null {
  const intent = String(value ?? '')
  return intentIds.has(intent as IntentId) ? intent as IntentId : null
}

export function intentGroupById(id: IntentGroupId): IntentGroup {
  const group = groupById.get(id)
  if (!group) throw new TypeError(`Unknown intent group: ${id}`)
  return group
}

export function groupForIntent(intent: IntentId): IntentGroup {
  const group = groupByIntent.get(intent)
  if (!group) throw new TypeError(`Intent is not mapped to a group: ${intent}`)
  return group
}

export interface SessionIntentGroupSource {
  intent?: string | null
  context?: {
    intent?: string | null
  } | null
}

/**
 * Project a persisted session to its visible intent group. Prefer the durable
 * intent context, then fall back to the runtime intent.
 */
export function intentGroupForSession(session: SessionIntentGroupSource): IntentGroupId {
  const intent = normalizeIntent(session.context?.intent)
  if (intent) return groupForIntent(intent).id

  switch (session.intent) {
    case 'natal':
      return 'natal'
    case 'divination':
      return 'divination'
    case 'fengshui':
      return 'fengshui'
    case 'naming':
      return 'naming'
    default:
      return 'natal'
  }
}

export function marketingIntentToGroup(intent: unknown): IntentGroupId | null {
  const normalized = normalizeIntent(intent)
  return normalized ? groupForIntent(normalized).id : null
}

export function bridgeIntent(input: IntentBridgeInput): IntentBridgeResult {
  const groupId = normalizeIntentGroup(input.groupId)
  if (!groupId) return { status: 'invalid', reason: 'unknown-group' }

  const group = intentGroupById(groupId)
  const authoritativeIntent = input.slotFormResult ?? normalizeIntent(input.seed)

  // No intent means the selected group itself is the routing decision. The
  // free-form question is consultation content and is never a routing signal.
  if (!authoritativeIntent) {
    return {
      status: 'ok',
      groupId,
      intent: group.baseIntent,
      fallbackUsed: true,
      reason: 'group-base',
    }
  }

  const intentGroup = groupByIntent.get(authoritativeIntent)
  if (!intentGroup || intentGroup.id !== groupId) {
    return {
      status: 'cross-group',
      groupId,
      suggestedGroupId: intentGroup?.id ?? groupId,
    }
  }

  if (!group.bridgeIntents.includes(authoritativeIntent)) {
    return {
      status: 'pending-backend',
      groupId,
      pendingIntent: authoritativeIntent,
    }
  }

  return {
    status: 'ok',
    groupId,
    intent: authoritativeIntent,
    fallbackUsed: false,
    reason: input.slotFormResult ? 'slot-form' : 'seed',
  }
}
