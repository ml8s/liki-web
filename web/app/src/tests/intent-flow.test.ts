import { describe, expect, it } from 'vitest'
import {
  intentGroups,
  bridgeIntent,
  intentGroupForSession,
  groupForIntent,
  marketingIntentToGroup,
  normalizeIntentGroup,
} from '@shared/domain/intent-flow'

describe('four-group flow contract', () => {
  it('defines exactly four user-facing practice groups', () => {
    expect(intentGroups.map(group => group.id)).toEqual([
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
  })

  it('maps every legacy marketing intent to one visible intent group', () => {
    expect([
      'natal',
      'liunian',
      'hepan',
      'huangli',
      'divination',
      'fengshui',
      'naming',
    ].map(marketingIntentToGroup)).toEqual([
      'natal',
      'natal',
      'natal',
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
    expect(marketingIntentToGroup('unknown')).toBeNull()
  })

  it('projects restored legacy sessions to their visible intent group', () => {
    expect(groupForIntent('huangli').id).toBe('natal')
    expect(groupForIntent('liunian').id).toBe('natal')
    expect(groupForIntent('divination').id).toBe('divination')
    expect(groupForIntent('fengshui').id).toBe('fengshui')
    expect(groupForIntent('naming').id).toBe('naming')
  })

  it('projects persisted sessions from durable context before intent', () => {
    expect(intentGroupForSession({
      intent: 'divination',
      context: { intent: 'huangli' },
    })).toBe('natal')

    expect(intentGroupForSession({ intent: 'natal' })).toBe('natal')
    expect(intentGroupForSession({ intent: 'divination' })).toBe('divination')
    expect(intentGroupForSession({ intent: 'fengshui' })).toBe('fengshui')
    expect(intentGroupForSession({ intent: 'naming' })).toBe('naming')
    expect(intentGroupForSession({})).toBe('natal')
  })

  it('rejects unknown groups without selecting an intent', () => {
    expect(normalizeIntentGroup('unknown')).toBeNull()
    expect(bridgeIntent({ groupId: 'unknown' })).toEqual({
      status: 'invalid',
      reason: 'unknown-group',
    })
  })

  it('uses the selected intent group base intent for a free-form consultation', () => {
    const cases = [
      ['natal', 'natal'],
      ['divination', 'divination'],
      ['fengshui', 'fengshui'],
      ['naming', 'naming'],
    ] as const

    for (const [groupId, intent] of cases) {
      expect(bridgeIntent({ groupId })).toMatchObject({
        status: 'ok',
        groupId,
        intent,
        fallbackUsed: true,
        reason: 'group-base',
      })
    }
  })

  it('uses explicit marketing and form seeds before the group base skill', () => {
    expect(
      bridgeIntent({
        groupId: 'natal',
        seed: 'liunian',
      }),
    ).toMatchObject({
      status: 'ok',
      intent: 'liunian',
      fallbackUsed: false,
      reason: 'seed',
    })

    expect(
      bridgeIntent({
        groupId: 'natal',
        slotFormResult: 'hepan',
      }),
    ).toMatchObject({
      status: 'ok',
      intent: 'hepan',
      fallbackUsed: false,
      reason: 'slot-form',
    })
  })

  it('never silently routes a seed outside the selected intent group', () => {
    expect(
      bridgeIntent({
        groupId: 'natal',
        seed: 'fengshui',
      }),
    ).toEqual({
      status: 'cross-group',
      groupId: 'natal',
      suggestedGroupId: 'fengshui',
    })

    expect(
      bridgeIntent({
        groupId: 'naming',
        seed: 'divination',
      }),
    ).toEqual({
      status: 'cross-group',
      groupId: 'naming',
      suggestedGroupId: 'divination',
    })
  })

  it('does not fake new huangli routing before backend ownership moves to natal', () => {
    expect(
      bridgeIntent({
        groupId: 'natal',
        seed: 'huangli',
      }),
    ).toEqual({
      status: 'pending-backend',
      groupId: 'natal',
      pendingIntent: 'huangli',
    })
  })
})
