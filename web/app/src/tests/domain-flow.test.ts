import { describe, expect, it } from 'vitest'
import {
  appDomains,
  bridgeLegacyIntent,
  domainForSession,
  domainForScenario,
  marketingScenarioToDomain,
  normalizeAppDomain,
} from '@shared/domain/flow'

describe('four-domain flow contract', () => {
  it('defines exactly four user-facing practice domains', () => {
    expect(appDomains.map(domain => domain.id)).toEqual([
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
  })

  it('maps every legacy marketing scenario to one visible domain', () => {
    expect([
      'natal',
      'liunian',
      'hepan',
      'huangli',
      'divination',
      'fengshui',
      'naming',
    ].map(marketingScenarioToDomain)).toEqual([
      'natal',
      'natal',
      'natal',
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
    expect(marketingScenarioToDomain('unknown')).toBeNull()
  })

  it('projects restored legacy sessions to their visible domain', () => {
    expect(domainForScenario('huangli').id).toBe('natal')
    expect(domainForScenario('liunian').id).toBe('natal')
    expect(domainForScenario('divination').id).toBe('divination')
    expect(domainForScenario('fengshui').id).toBe('fengshui')
    expect(domainForScenario('naming').id).toBe('naming')
  })

  it('projects persisted sessions from durable context before expert', () => {
    expect(domainForSession({
      expert: 'divination',
      context: { scenario: 'huangli' },
    })).toBe('natal')

    expect(domainForSession({ expert: 'natal' })).toBe('natal')
    expect(domainForSession({ expert: 'divination' })).toBe('divination')
    expect(domainForSession({ expert: 'fengshui' })).toBe('fengshui')
    expect(domainForSession({ expert: 'naming' })).toBe('naming')
    expect(domainForSession({})).toBe('natal')
  })

  it('rejects unknown domains without selecting an expert', () => {
    expect(normalizeAppDomain('unknown')).toBeNull()
    expect(bridgeLegacyIntent({ domainId: 'unknown' })).toEqual({
      status: 'invalid',
      reason: 'unknown-domain',
    })
  })

  it('uses the selected domain base skill for a free-form consultation', () => {
    const cases = [
      ['natal', 'natal', 'natal'],
      ['divination', 'divination', 'divination'],
      ['fengshui', 'fengshui', 'fengshui'],
      ['naming', 'naming', 'naming'],
    ] as const

    for (const [domainId, skill, expert] of cases) {
      expect(bridgeLegacyIntent({ domainId })).toMatchObject({
        status: 'ok',
        domainId,
        skill,
        expert,
        fallbackUsed: true,
        reason: 'domain-base',
      })
    }
  })

  it('uses explicit marketing and form seeds before the domain base skill', () => {
    expect(
      bridgeLegacyIntent({
        domainId: 'natal',
        seed: 'liunian',
      }),
    ).toMatchObject({
      status: 'ok',
      skill: 'liunian',
      expert: 'natal',
      fallbackUsed: false,
      reason: 'seed',
    })

    expect(
      bridgeLegacyIntent({
        domainId: 'natal',
        slotFormResult: 'hepan',
      }),
    ).toMatchObject({
      status: 'ok',
      skill: 'hepan',
      expert: 'natal',
      fallbackUsed: false,
      reason: 'slot-form',
    })
  })

  it('never silently routes a seed outside the selected domain', () => {
    expect(
      bridgeLegacyIntent({
        domainId: 'natal',
        seed: 'fengshui',
      }),
    ).toEqual({
      status: 'cross-domain',
      domainId: 'natal',
      suggestedDomainId: 'fengshui',
    })

    expect(
      bridgeLegacyIntent({
        domainId: 'naming',
        seed: 'divination',
      }),
    ).toEqual({
      status: 'cross-domain',
      domainId: 'naming',
      suggestedDomainId: 'divination',
    })
  })

  it('does not fake new huangli routing before backend ownership moves to natal', () => {
    expect(
      bridgeLegacyIntent({
        domainId: 'natal',
        seed: 'huangli',
      }),
    ).toEqual({
      status: 'pending-backend',
      domainId: 'natal',
      legacySkill: 'huangli',
    })
  })
})
