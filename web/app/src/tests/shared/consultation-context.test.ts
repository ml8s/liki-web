import { describe, expect, it } from 'vitest'
import {
  sameConsultationBirth,
  sameConsultationLayout,
  type ConsultationBirth,
  type ConsultationLayout,
} from '@shared/domain/consultation'

const birth: ConsultationBirth = {
  gender: 'female',
  date: '1990-05-12',
  time: '08:30',
  location: { longitude: 121.47, timezone: 'Asia/Shanghai' },
}

describe('sameConsultationBirth', () => {
  it('matches identical birth projections', () => {
    expect(sameConsultationBirth(birth, { ...birth })).toBe(true)
  })

  it('rejects any changed field', () => {
    expect(sameConsultationBirth(birth, { ...birth, gender: 'male' })).toBe(false)
    expect(sameConsultationBirth(birth, { ...birth, date: '1990-05-13' })).toBe(false)
    expect(sameConsultationBirth(birth, { ...birth, time: '09:30' })).toBe(false)
    expect(sameConsultationBirth(birth, {
      ...birth,
      location: { longitude: 0, timezone: 'UTC' },
    })).toBe(false)
  })

  it('requires both sides', () => {
    expect(sameConsultationBirth(undefined, birth)).toBe(false)
    expect(sameConsultationBirth(birth, undefined)).toBe(false)
    expect(sameConsultationBirth(undefined, undefined)).toBe(false)
  })
})

describe('sameConsultationLayout', () => {
  const layout: ConsultationLayout = {
    zuo_shan: 3,
    period_date: '2026-01-01',
    door_gua: '坎',
    master_gua: '离',
    stove_gua: '震',
  }

  it('matches identical layouts', () => {
    expect(sameConsultationLayout(layout, { ...layout })).toBe(true)
  })

  it('rejects any changed field', () => {
    expect(sameConsultationLayout(layout, { ...layout, zuo_shan: 7 })).toBe(false)
    expect(sameConsultationLayout(layout, { ...layout, period_date: '2026-02-01' })).toBe(false)
    expect(sameConsultationLayout(layout, { ...layout, door_gua: '坤' })).toBe(false)
  })

  it('requires both sides', () => {
    expect(sameConsultationLayout(undefined, layout)).toBe(false)
    expect(sameConsultationLayout(layout, undefined)).toBe(false)
  })
})
