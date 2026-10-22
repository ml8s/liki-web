import type { BirthLocation } from '@shared/domain/location'
import type { BirthInfo } from '@shared/utils/paipan'
import type { RegionFieldLabels } from './region-field'

export type BirthGender = 'female' | 'male'

export interface BirthFieldLabels {
  gender: string
  female: string
  male: string
  genderPlaceholder: string
  birthDate: string
  selectDate: string
  birthTime: string
  timePlaceholder: string
  region: RegionFieldLabels
}

/** Complete chart input; an incomplete form is simply not a chart input. */
export interface BirthFieldsValue {
  gender: BirthGender | ''
  birth_date: string
  birth_time: string
  location: BirthLocation | null
}

type BirthLabelTranslate = (key: string, values?: Record<string, unknown>) => string

export function createBirthFieldLabels(t: BirthLabelTranslate): BirthFieldLabels {
  return {
    gender: t('profiles.form.gender'),
    female: t('profiles.female'),
    male: t('profiles.male'),
    genderPlaceholder: t('profiles.form.genderPlaceholder'),
    birthDate: t('profiles.form.birthDate'),
    selectDate: t('profiles.form.selectDate'),
    birthTime: t('profiles.form.birthTime'),
    timePlaceholder: t('profiles.form.timePlaceholder'),
    region: {
      place: t('profiles.form.birthPlace'),
      placePlaceholder: t('profiles.form.birthPlacePlaceholder'),
      placeResolving: t('profiles.form.placeResolving'),
      placeAmbiguous: t('profiles.form.placeAmbiguous'),
      placePick: t('profiles.form.placePick'),
      placeNotFound: t('profiles.form.placeNotFound'),
      placeMatched: t('profiles.form.placeMatched'),
      longitude: t('profiles.form.longitude'),
      longitudeHint: t('profiles.form.longitudeHint'),
      timezone: t('profiles.form.timezone'),
      timezonePlaceholder: t('profiles.form.timezonePlaceholder'),
      timezoneHint: t('profiles.form.timezoneHint'),
    },
  }
}

export function createBirthFieldsValue(): BirthFieldsValue {
  return {
    gender: '',
    birth_date: '',
    birth_time: '',
    location: null,
  }
}

export function isBirthInputReady(value: BirthFieldsValue): boolean {
  return (value.gender === 'female' || value.gender === 'male')
    && /^\d{4}-\d{2}-\d{2}$/.test(value.birth_date)
    && /^\d{2}:\d{2}$/.test(value.birth_time)
    && value.location !== null
}

/** Adapter at the component boundary: shared form state → domain BirthInfo. */
export function birthInfoFromFields(value: BirthFieldsValue): BirthInfo {
  if (!isBirthInputReady(value) || !value.location) {
    throw new TypeError('Complete birth information is required')
  }
  const gender = value.gender
  if (gender !== 'female' && gender !== 'male') {
    throw new TypeError('A valid gender is required')
  }
  return {
    gender,
    birth_date: value.birth_date,
    birth_time: value.birth_time,
    birth_place: value.location.place,
    birth_longitude: value.location.longitude,
    birth_timezone: value.location.timezone,
  }
}
