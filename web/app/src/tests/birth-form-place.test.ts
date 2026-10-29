import { describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { ElAutocomplete } from 'element-plus'
import BirthForm from '@/components/birth-profiles/BirthForm.vue'
import { resetCityIndexCache } from '@shared/utils/citySearch'

const toolOk = (data: unknown) =>
  new Response(
    JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [], structuredContent: data },
    }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  )

function mountForm(initial: Record<string, unknown> = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(BirthForm, { props: { initial }, global: { plugins: [pinia] } })
}

const longitudeValue = (wrapper: ReturnType<typeof mountForm>) =>
  (wrapper.find('input[name="birth-longitude"]').element as HTMLInputElement).value

async function settleBlur() {
  await flushPromises()
  await new Promise(resolve => setTimeout(resolve, 0))
  await flushPromises()
}

function openManualFields(wrapper: ReturnType<typeof mountForm>) {
  return wrapper.find('.region-manual-toggle').trigger('click')
}

describe('BirthForm place resolution', () => {
  beforeEach(() => { resetCityIndexCache() })

  it('auto-fills longitude for a unique candidate', async () => {
    const index = { counties: [], cities: [{ name: '上海市', region: '上海市', lng: 121.4737, lat: 31.2304, tz: 'Asia/Shanghai' }], world: [] }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => Promise.resolve(
      String(url).includes('cities_data.json')
        ? new Response(JSON.stringify(index), { status: 200, headers: { 'Content-Type': 'application/json' } })
        : toolOk({ candidates: [] }),
    )))
    const wrapper = mountForm()
    await wrapper.find('input[name="birth-place"]').setValue('上海')
    await wrapper.find('input[name="birth-place"]').trigger('blur')
    await settleBlur()
    await openManualFields(wrapper)
    expect(longitudeValue(wrapper)).toBe('121.4737')
  })

  it('lets the user pick among ambiguous candidates', async () => {
    const index = { counties: [{ name: '长沙县', region: '湖南省', lng: 113.075, lat: 28.2502, tz: 'Asia/Shanghai' }], cities: [{ name: '长沙市', region: '湖南省', lng: 113.0267, lat: 28.1886, tz: 'Asia/Shanghai' }], world: [] }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => Promise.resolve(
      String(url).includes('cities_data.json')
        ? new Response(JSON.stringify(index), { status: 200, headers: { 'Content-Type': 'application/json' } })
        : toolOk({ candidates: [] }),
    )))
    const wrapper = mountForm()
    await wrapper.find('input[name="birth-place"]').setValue('长沙')
    await new Promise(resolve => setTimeout(resolve, 130))
    await flushPromises()
    await wrapper.find('input[name="birth-place"]').trigger('blur')
    await settleBlur()
    const autocomplete = wrapper.findComponent(ElAutocomplete)
    const componentVm = (autocomplete as unknown as { vm?: unknown }).vm
    const exposed = componentVm as {
      suggestions: Array<{ candidate: unknown }>
      handleSelect: (item: unknown) => void
    }
    const choice = exposed.suggestions[1]
    expect(choice).toBeTruthy()
    exposed.handleSelect(choice)
    await flushPromises()
    await openManualFields(wrapper)
    expect(longitudeValue(wrapper)).toBe('113.0750')
  })

  it('does not infer a location for legacy v0 profiles', async () => {
    const wrapper = mountForm({ nickname: '妈妈', birth: { location: { place: '上海', location_version: 0 } } })
    await flushPromises()
    await openManualFields(wrapper)
    expect(longitudeValue(wrapper)).toBe('')
    expect((wrapper.find('input[name="birth-place"]').element as HTMLInputElement).value).toBe('')
  })
})
