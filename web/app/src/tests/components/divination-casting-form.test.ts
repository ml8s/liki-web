import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import { readStoredDivinationCastingValue } from '@shared/components/divination/types'

describe('DivinationCastingForm', () => {
  it('emits a public qigua payload after validation', async () => {
    const wrapper = mount(DivinationCastingForm, { props: { variant: 'public-qigua' } })
    await wrapper.find('input[name="divination-question"]').setValue('這個項目能成嗎')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toEqual([[
      {
        question: '這個項目能成嗎',
        castingMode: 'auto',
        location: null,
      },
    ]])
  })

  it('does not submit without a question', async () => {
    const wrapper = mount(DivinationCastingForm, { props: { variant: 'public-qigua' } })
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('restores and exposes the full casting context', () => {
    const wrapper = mount(DivinationCastingForm, { props: { variant: 'full' } })
    wrapper.vm.restore({
      question: '這個項目能成嗎',
      casting_mode: 'auto',
      casting_place: '上海市',
      casting_longitude: 121.4737,
      casting_timezone: 'Asia/Shanghai',
    })

    expect(wrapper.vm.getContext()).toEqual({
      question: '這個項目能成嗎',
      castingMode: 'auto',
      location: {
        place: '上海市',
        longitude: 121.4737,
        timezone: 'Asia/Shanghai',
        source: 'saved',
      },
    })
    expect(readStoredDivinationCastingValue({ question: '舊問題' })).toEqual({
      question: '舊問題',
      castingMode: 'auto',
      location: null,
    })
  })
})
