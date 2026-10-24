import type { BirthGender, BirthInput } from '@shared/domain/birth'
import type { BirthLocation } from '@shared/domain/location'
import type { RegionFieldLabels } from './region-field'

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
  date: string
  time: string
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
      longitudeInvalid: t('profiles.form.longitudeInvalid'),
      timezone: t('profiles.form.timezone'),
      timezonePlaceholder: t('profiles.form.timezonePlaceholder'),
      timezoneHint: t('profiles.form.timezoneHint'),
      timezoneInvalid: t('profiles.form.timezoneInvalid'),
      manualToggle: t('profiles.form.manualToggle'),
      kindCity: t('profiles.form.kindCity'),
      kindCounty: t('profiles.form.kindCounty'),
      kindWorld: t('profiles.form.kindWorld'),
    },
  }
}

export function createBirthFieldsValue(): BirthFieldsValue {
  return {
    gender: '',
    date: '',
    time: '',
    location: null,
  }
}

export function isBirthInputReady(value: BirthFieldsValue): boolean {
  return (value.gender === 'female' || value.gender === 'male')
    && /^\d{4}-\d{2}-\d{2}$/.test(value.date)
    && /^\d{2}:\d{2}$/.test(value.time)
    && value.location !== null
}

/** Adapter at the component boundary: shared form state → domain BirthInput. */
export function birthInputFromFields(value: BirthFieldsValue): BirthInput {
  if (!isBirthInputReady(value) || !value.location) {
    throw new TypeError('Complete birth information is required')
  }
  const gender = value.gender
  if (gender !== 'female' && gender !== 'male') {
    throw new TypeError('A valid gender is required')
  }
  return {
    gender,
    date: value.date,
    time: value.time,
    location: value.location,
  }
}
