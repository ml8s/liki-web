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
