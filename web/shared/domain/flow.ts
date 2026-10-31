import domainContract from '../config/domains.json'
import intentContract from '../config/intents.json'

export type AppDomainId = 'natal' | 'divination' | 'fengshui' | 'naming'
export type LegacyIntentId =
  | 'natal'
  | 'liunian'
  | 'hepan'
  | 'divination'
  | 'huangli'
  | 'fengshui'
  | 'naming'
export type IntentId = 'natal' | 'divination' | 'fengshui' | 'naming'
export type DomainProfilePolicy = 'required' | 'optional' | 'partial'

export interface AppDomain {
  id: AppDomainId
  labelKey: string
  baseSkill: LegacyIntentId
  acceptedSeeds: readonly LegacyIntentId[]
  /** Legacy skills the frontend may create in this domain before backend P4. */
  bridgeSkills: readonly LegacyIntentId[]
  hybridDefaultPath: string
  profilePolicy: DomainProfilePolicy
}

export interface LegacyIntentBridgeInput {
  domainId: string
  personId?: string
  seed?: string | null
  slotFormResult?: LegacyIntentId | null
}

export type LegacyIntentBridgeResult =
  | {
      status: 'ok'
      domainId: AppDomainId
      skill: LegacyIntentId
      fallbackUsed: boolean
      reason: 'seed' | 'slot-form' | 'domain-base'
    }
  | {
      status: 'cross-domain'
      domainId: AppDomainId
      suggestedDomainId: AppDomainId
    }
  | {
      status: 'pending-backend'
      domainId: AppDomainId
      legacySkill: LegacyIntentId
    }
  | {
      status: 'invalid'
      reason: 'unknown-domain' | 'empty-routing-input'
    }

export const appDomains: readonly AppDomain[] = domainContract.domains.map(domain => ({
  id: domain.id as AppDomainId,
  labelKey: domain.labelKey,
  baseSkill: domain.baseSkill as LegacyIntentId,
  acceptedSeeds: domain.acceptedSeeds as LegacyIntentId[],
  bridgeSkills: domain.bridgeSkills as LegacyIntentId[],
  hybridDefaultPath: domain.hybridDefaultPath,
  profilePolicy: domain.profilePolicy as DomainProfilePolicy,
}))

const domainBySeed = new Map<LegacyIntentId, AppDomain>()
for (const domain of appDomains) {
  for (const seed of domain.acceptedSeeds) domainBySeed.set(seed, domain)
}

const domainById = new Map<AppDomainId, AppDomain>(
  appDomains.map(domain => [domain.id, domain]),
)

const legacyIntentIds = new Set(
  intentContract.intents.map(intent => intent.id as LegacyIntentId),
)

export function normalizeAppDomain(value: unknown): AppDomainId | null {
  const id = String(value ?? '')
  return domainById.has(id as AppDomainId) ? id as AppDomainId : null
}

export function normalizeLegacyIntent(value: unknown): LegacyIntentId | null {
  const intent = String(value ?? '')
  return legacyIntentIds.has(intent as LegacyIntentId)
    ? intent as LegacyIntentId
    : null
}

export function appDomainById(id: AppDomainId): AppDomain {
  const domain = domainById.get(id)
  if (!domain) throw new TypeError(`Unknown app domain: ${id}`)
  return domain
}

export function domainForIntent(intent: LegacyIntentId): AppDomain {
  const domain = domainBySeed.get(intent)
  if (!domain) throw new TypeError(`Intent is not mapped to a domain: ${intent}`)
  return domain
}

export interface SessionDomainSource {
  intent?: string | null
  context?: {
    intent?: string | null
  } | null
}

/**
 * Project a persisted session to its visible domain. Prefer the durable
 * intent context, then fall back to the runtime intent. This lets legacy
 * huangli sessions surface under 命理 even while their intent is divination.
 */
export function domainForSession(session: SessionDomainSource): AppDomainId {
  const intent = normalizeLegacyIntent(session.context?.intent)
  if (intent) return domainForIntent(intent).id

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

export function marketingIntentToDomain(intent: unknown): AppDomainId | null {
  const normalized = normalizeLegacyIntent(intent)
  return normalized ? domainForIntent(normalized).id : null
}

export function bridgeLegacyIntent(input: LegacyIntentBridgeInput): LegacyIntentBridgeResult {
  const domainId = normalizeAppDomain(input.domainId)
  if (!domainId) return { status: 'invalid', reason: 'unknown-domain' }

  const domain = appDomainById(domainId)
  const authoritativeSeed = input.slotFormResult ?? normalizeLegacyIntent(input.seed)

  // No seed means the selected domain itself is the routing decision. The
  // free-form question is consultation content and is never a routing signal.
  if (!authoritativeSeed) {
    return {
      status: 'ok',
      domainId,
      skill: domain.baseSkill,
      fallbackUsed: true,
      reason: 'domain-base',
    }
  }

  const seedDomain = domainBySeed.get(authoritativeSeed)
  if (!seedDomain || seedDomain.id !== domainId) {
    return {
      status: 'cross-domain',
      domainId,
      suggestedDomainId: seedDomain?.id ?? domainId,
    }
  }

  if (!domain.bridgeSkills.includes(authoritativeSeed)) {
    return {
      status: 'pending-backend',
      domainId,
      legacySkill: authoritativeSeed,
    }
  }

  return {
    status: 'ok',
    domainId,
    skill: authoritativeSeed,
    fallbackUsed: false,
    reason: input.slotFormResult ? 'slot-form' : 'seed',
  }
}
