import { isValidBirthLocation, type BirthLocation } from '@shared/domain/location'

export type DivinationCastingLocationSource = BirthLocation['source']

export interface DivinationCastingFormValue {
  question: string
  castingMode: 'auto'
  location: BirthLocation | null
}

export function createDivinationCastingFormValue(): DivinationCastingFormValue {
  return {
    question: '',
    castingMode: 'auto',
    location: null,
  }
}

export function readStoredDivinationCastingValue(input: unknown): DivinationCastingFormValue {
  if (typeof input !== 'object' || input === null) return createDivinationCastingFormValue()
  const source = input as Record<string, unknown>
  const question = typeof source.question === 'string' ? source.question : ''
  const location = source.location
  if (isValidBirthLocation(location)) {
    return {
      question,
      castingMode: 'auto',
      location: { ...location, source: 'saved' },
    }
  }

  // Legacy records did not persist timezone; retain display facts without
  // pretending they are sufficient for a new chart call.
  const longitude = Number(source.casting_longitude)
  const place = typeof source.casting_place === 'string' ? source.casting_place : ''
  const timezone = typeof source.casting_timezone === 'string' ? source.casting_timezone : ''
  const restored = {
    place,
    longitude: Number.isFinite(longitude) ? longitude : null,
    timezone,
    source: 'saved' as const,
  }
  return {
    question,
    castingMode: 'auto',
    location: restored.longitude !== null && restored.timezone ? restored as BirthLocation : null,
  }
}
