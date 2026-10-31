import { describe, expect, it } from 'vitest'
import {
  appDomains,
  bridgeLegacyIntent,
  domainForSession,
  domainForIntent,
  marketingIntentToDomain,
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

  it('maps every legacy marketing intent to one visible domain', () => {
    expect([
      'natal',
      'liunian',
      'hepan',
      'huangli',
      'divination',
      'fengshui',
      'naming',
    ].map(marketingIntentToDomain)).toEqual([
      'natal',
      'natal',
      'natal',
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
    expect(marketingIntentToDomain('unknown')).toBeNull()
  })

  it('projects restored legacy sessions to their visible domain', () => {
    expect(domainForIntent('huangli').id).toBe('natal')
    expect(domainForIntent('liunian').id).toBe('natal')
    expect(domainForIntent('divination').id).toBe('divination')
    expect(domainForIntent('fengshui').id).toBe('fengshui')
    expect(domainForIntent('naming').id).toBe('naming')
  })

  it('projects persisted sessions from durable context before intent', () => {
    expect(domainForSession({
      intent: 'divination',
      context: { intent: 'huangli' },
    })).toBe('natal')

    expect(domainForSession({ intent: 'natal' })).toBe('natal')
    expect(domainForSession({ intent: 'divination' })).toBe('divination')
    expect(domainForSession({ intent: 'fengshui' })).toBe('fengshui')
    expect(domainForSession({ intent: 'naming' })).toBe('naming')
    expect(domainForSession({})).toBe('natal')
  })

  it('rejects unknown domains without selecting an intent', () => {
    expect(normalizeAppDomain('unknown')).toBeNull()
    expect(bridgeLegacyIntent({ domainId: 'unknown' })).toEqual({
      status: 'invalid',
      reason: 'unknown-domain',
    })
  })

  it('uses the selected domain base skill for a free-form consultation', () => {
    const cases = [
      ['natal', 'natal'],
      ['divination', 'divination'],
      ['fengshui', 'fengshui'],
      ['naming', 'naming'],
    ] as const

    for (const [domainId, skill] of cases) {
      expect(bridgeLegacyIntent({ domainId })).toMatchObject({
        status: 'ok',
        domainId,
        skill,
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
