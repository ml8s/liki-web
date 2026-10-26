import type { BirthFieldsValue } from '../components/form/birth-fields'
import type { LiuyaoCasting } from '../api/engine/liuyao'
import type { BirthLocation } from './location'

export type ConsultationScenario =
  | 'natal'
  | 'liunian'
  | 'hepan'
  | 'divination'
  | 'huangli'
  | 'fengshui'
  | 'naming'

export interface ConsultationBirth {
  gender: 'female' | 'male'
  date: string
  time: string
  location: Pick<BirthLocation, 'longitude' | 'timezone'>
}

export interface ConsultationLocation {
  longitude: number
  timezone: string
}

export interface ConsultationLayout {
  zuo_shan: number
  period_date: string
  door_gua?: string
  master_gua?: string
  stove_gua?: string
}

export interface ConsultationContext {
  version: 1
  scenario: ConsultationScenario
  birth?: ConsultationBirth
  second_birth?: ConsultationBirth
  birth_year?: number
  gender?: 'female' | 'male'
  year?: number
  question?: string
  casting?: LiuyaoCasting
  casting_instant?: string
  location?: ConsultationLocation
  event?: string
  start_date?: string
  end_date?: string
  layout?: ConsultationLayout
  /** He Pan must affirmatively carry this server-enforced domain fact. */
  third_party_consent?: true
}

export function consultationBirthFromFields(value: BirthFieldsValue): ConsultationBirth {
  if (!value.location) {
    throw new TypeError('Complete birth location is required')
  }
  if (value.gender !== 'female' && value.gender !== 'male') {
    throw new TypeError('A valid gender is required')
  }
  return {
    gender: value.gender,
    date: value.date,
    time: value.time,
    location: {
      longitude: value.location.longitude,
      timezone: value.location.timezone,
    },
  }
}

/**
 * Strict equality for the birth projection carried in a consultation context.
 * Used by follow-up matching: a session matches a focused form only when the
 * birth inputs are identical, so a changed profile never silently reuses an
 * old expert context.
 */
export function sameConsultationBirth(
  left: ConsultationBirth | undefined,
  right: ConsultationBirth | undefined,
): boolean {
  if (!left || !right) return false
  return left.gender === right.gender
    && left.date === right.date
    && left.time === right.time
    && left.location.longitude === right.location.longitude
    && left.location.timezone === right.location.timezone
}

/**
 * Strict equality for the Fengshui layout projection. Same matching rule as
 * `sameConsultationBirth`: identical layout inputs only.
 */
export function sameConsultationLayout(
  left: ConsultationLayout | undefined,
  right: ConsultationLayout | undefined,
): boolean {
  if (!left || !right) return false
  return left.zuo_shan === right.zuo_shan
    && left.period_date === right.period_date
    && left.door_gua === right.door_gua
    && left.master_gua === right.master_gua
    && left.stove_gua === right.stove_gua
}
