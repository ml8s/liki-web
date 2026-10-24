import { mount } from '@vue/test-utils'
import BirthFields from '@shared/components/form/BirthFields.vue'
import {
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'

const labels: BirthFieldLabels = {
  gender: 'Gender',
  female: 'Female',
  male: 'Male',
  genderPlaceholder: 'Gender',
  birthDate: 'Birth date',
  selectDate: 'Select date',
  birthTime: 'Birth time',
  timePlaceholder: '06:00',
  region: {
    place: 'Place',
    placePlaceholder: 'Place',
    placeResolving: 'Resolving',
    placeAmbiguous: 'Ambiguous',
    placePick: 'Choose',
    placeNotFound: 'Not found',
    placeMatched: 'Matched {longitude}',
    longitude: 'Longitude',
    longitudeHint: 'Longitude hint',
  longitudeInvalid: 'Invalid longitude',
    timezone: 'Timezone',
    timezonePlaceholder: 'Asia/Shanghai',
    timezoneHint: 'Timezone hint',
  timezoneInvalid: 'Invalid timezone',
    manualToggle: 'Enter manually',
    kindCity: 'City',
    kindCounty: 'County',
    kindWorld: 'International',
  },
}

const base: BirthFieldsValue = createBirthFieldsValue()

describe('BirthFields', () => {
  it('emits one merged update when a known-time field changes', async () => {
    const wrapper = mount(BirthFields, { props: { value: base, labels } })
    await wrapper.find('input[value="female"]').setValue(true)
    const updates = wrapper.emitted('update:patch')
    expect(updates).toHaveLength(1)
    expect(updates?.[0]?.[0]).toEqual({ gender: 'female' })
  })

  it('always renders a Gregorian date picker and has no lunar input', async () => {
    const wrapper = mount(BirthFields, { props: { value: base, labels } })
    expect(wrapper.find('input[name="birth-lunar-date"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Lunar')
  })

  it('requires complete time plus an atomic location for deterministic charts', () => {
    const location = {
      place: '上海', longitude: 121.4737, timezone: 'Asia/Shanghai', source: 'city' as const,
    }
    const exact = { ...base, gender: 'female' as const, date: '1958-05-12', time: '06:00', location }
    expect(isBirthInputReady(exact)).toBe(true)
    expect(isBirthInputReady({ ...exact, location: null })).toBe(false)
  })
})
