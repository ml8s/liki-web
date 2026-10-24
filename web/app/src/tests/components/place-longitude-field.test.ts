import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { ElForm } from 'element-plus'
import RegionAutocompleteField from '@shared/components/form/fields/RegionAutocompleteField.vue'
import type { BirthLocation } from '@shared/domain/location'
import type { RegionFieldLabels } from '@shared/components/form/region-field'
import { resetCityIndexCache } from '@shared/utils/citySearch'

const labels: RegionFieldLabels = {
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
}

const shanghai: BirthLocation = {
  place: '上海市',
  longitude: 121.4737,
  timezone: 'Asia/Shanghai',
  source: 'city',
}

function mountField(initial: BirthLocation | null = null) {
  const location = ref<BirthLocation | null>(initial)
  const wrapper = mount({
    components: { ElForm, RegionAutocompleteField },
    setup: () => ({ location, labels }),
    template: `<el-form :model="{}"><RegionAutocompleteField v-model:location="location" :labels="labels" /></el-form>`,
  })
  return { wrapper, location }
}

describe('RegionAutocompleteField', () => {
  beforeEach(() => resetCityIndexCache())

  it('keeps an externally restored longitude/timezone pair', async () => {
    const { location } = mountField(shanghai)
    expect(location.value).toEqual(shanghai)
  })

  it('resolves a place into an atomic location', async () => {
    const index = {
      counties: [],
      cities: [{
        id: 'CN-310000', name: '上海市', region: '上海市', lng: 121.4737, lat: 31.2304, tz: 'Asia/Shanghai',
      }],
      world: [],
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify(index), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    ))
    const { wrapper, location } = mountField()
    const input = wrapper.find('input[name="birth-place"]')
    await input.setValue('上海')
    expect(location.value).toBeNull()
    await input.trigger('blur')
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(location.value).toMatchObject({
      place: '上海市',
      place_id: 'CN-310000',
      longitude: 121.4737,
      timezone: 'Asia/Shanghai',
      source: 'city',
    })
  })

  it('invalidates the atomic location when the user edits the place', async () => {
    const { wrapper, location } = mountField(shanghai)
    await wrapper.find('input[name="birth-place"]').setValue('杭州')
    expect(location.value).toBeNull()
    await wrapper.find('.region-manual-toggle').trigger('click')
    // A city-derived timezone must not silently pair with a new manual place.
    expect((wrapper.find('input[name="birth-timezone"]').element as HTMLInputElement).value).toBe('')
  })
})
