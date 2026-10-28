import domainContract from '../config/domains.json'
import scenarioContract from '../config/scenarios.json'

export type AppDomainId = 'natal' | 'divination' | 'fengshui' | 'naming'
export type LegacyScenarioId =
  | 'natal'
  | 'liunian'
  | 'hepan'
  | 'divination'
  | 'huangli'
  | 'fengshui'
  | 'naming'
export type ExpertId = 'natal' | 'divination' | 'fengshui' | 'naming'
export type DomainProfilePolicy = 'required' | 'optional' | 'partial'

export interface AppDomain {
  id: AppDomainId
  labelKey: string
  baseSkill: LegacyScenarioId
  acceptedSeeds: readonly LegacyScenarioId[]
  /** Legacy skills the frontend may create in this domain before backend P4. */
  bridgeSkills: readonly LegacyScenarioId[]
  hybridDefaultPath: string
  profilePolicy: DomainProfilePolicy
}

export interface LegacyIntentBridgeInput {
  domainId: string
  personId?: string
  seed?: string | null
  slotFormResult?: LegacyScenarioId | null
}

export type LegacyIntentBridgeResult =
  | {
      status: 'ok'
      domainId: AppDomainId
      skill: LegacyScenarioId
      expert: ExpertId
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
      legacySkill: LegacyScenarioId
    }
  | {
      status: 'invalid'
      reason: 'unknown-domain' | 'empty-routing-input'
    }

export const appDomains: readonly AppDomain[] = domainContract.domains.map(domain => ({
  id: domain.id as AppDomainId,
  labelKey: domain.labelKey,
  baseSkill: domain.baseSkill as LegacyScenarioId,
  acceptedSeeds: domain.acceptedSeeds as LegacyScenarioId[],
  bridgeSkills: domain.bridgeSkills as LegacyScenarioId[],
  hybridDefaultPath: domain.hybridDefaultPath,
  profilePolicy: domain.profilePolicy as DomainProfilePolicy,
}))

const legacyScenarios = scenarioContract.scenarios as readonly {
  id: LegacyScenarioId
  expert: ExpertId
}[]

const expertByScenario = new Map<LegacyScenarioId, ExpertId>(
  legacyScenarios.map(scenario => [scenario.id, scenario.expert]),
)

const domainBySeed = new Map<LegacyScenarioId, AppDomain>()
for (const domain of appDomains) {
  for (const seed of domain.acceptedSeeds) domainBySeed.set(seed, domain)
}

const domainById = new Map<AppDomainId, AppDomain>(
  appDomains.map(domain => [domain.id, domain]),
)

const legacyScenarioIds = new Set(expertByScenario.keys())

export function normalizeAppDomain(value: unknown): AppDomainId | null {
  const id = String(value ?? '')
  return domainById.has(id as AppDomainId) ? id as AppDomainId : null
}

export function normalizeLegacyScenario(value: unknown): LegacyScenarioId | null {
  const scenario = String(value ?? '')
  return legacyScenarioIds.has(scenario as LegacyScenarioId)
    ? scenario as LegacyScenarioId
    : null
}

export function appDomainById(id: AppDomainId): AppDomain {
  const domain = domainById.get(id)
  if (!domain) throw new TypeError(`Unknown app domain: ${id}`)
  return domain
}

export function domainForScenario(scenario: LegacyScenarioId): AppDomain {
  const domain = domainBySeed.get(scenario)
  if (!domain) throw new TypeError(`Scenario is not mapped to a domain: ${scenario}`)
  return domain
}

export interface SessionDomainSource {
  expert?: string | null
  context?: {
    scenario?: string | null
  } | null
}

/**
 * Project a persisted session to its visible domain. Prefer the durable
 * scenario context, then fall back to the runtime expert. This lets legacy
 * huangli sessions surface under 命理 even while their expert is divination.
 */
export function domainForSession(session: SessionDomainSource): AppDomainId {
  const scenario = normalizeLegacyScenario(session.context?.scenario)
  if (scenario) return domainForScenario(scenario).id

  switch (session.expert) {
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

export function marketingScenarioToDomain(scenario: unknown): AppDomainId | null {
  const normalized = normalizeLegacyScenario(scenario)
  return normalized ? domainForScenario(normalized).id : null
}

export function bridgeLegacyIntent(input: LegacyIntentBridgeInput): LegacyIntentBridgeResult {
  const domainId = normalizeAppDomain(input.domainId)
  if (!domainId) return { status: 'invalid', reason: 'unknown-domain' }

  const domain = appDomainById(domainId)
  const authoritativeSeed = input.slotFormResult ?? normalizeLegacyScenario(input.seed)

  // No seed means the selected domain itself is the routing decision. The
  // free-form question is consultation content and is never a routing signal.
  if (!authoritativeSeed) {
    const expert = expertByScenario.get(domain.baseSkill)
    if (!expert) return { status: 'invalid', reason: 'unknown-domain' }
    return {
      status: 'ok',
      domainId,
      skill: domain.baseSkill,
      expert,
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

  const expert = expertByScenario.get(authoritativeSeed)
  if (!expert) return { status: 'invalid', reason: 'unknown-domain' }
  return {
    status: 'ok',
    domainId,
    skill: authoritativeSeed,
    expert,
    fallbackUsed: false,
    reason: input.slotFormResult ? 'slot-form' : 'seed',
  }
}
