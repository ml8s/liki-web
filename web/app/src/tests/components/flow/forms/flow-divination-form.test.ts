import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import FlowDivinationForm from '@/components/flow/forms/FlowDivinationForm.vue'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'

const savedLocation: DivinationCastingFormValue['location'] = {
  place: '上海市',
  longitude: 121.4737,
  timezone: 'Asia/Shanghai',
  source: 'saved',
}

describe('FlowDivinationForm', () => {
  it('submits a validated typed casting value', async () => {
    const wrapper = mount(FlowDivinationForm)

    wrapper.vm.restore({ question: '', location: savedLocation })
    await wrapper.vm.$nextTick()
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.emitted('submit')).toBeUndefined()

    await wrapper.find('textarea[name="divination-question"]').setValue('这周适合签约吗？')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toEqual([[
      {
        question: '这周适合签约吗？',
        castingMode: 'auto',
        location: savedLocation,
      },
    ]])
  })

  it('blocks the shared form while loading and suppresses its submit event', async () => {
    const wrapper = mount(FlowDivinationForm, {
      props: { loading: true },
    })
    const form = wrapper.find('form')

    expect(wrapper.find('.flow-divination-form').attributes('aria-busy')).toBe('true')
    expect(wrapper.find('.flow-divination-form').attributes('inert')).toBeDefined()
    expect(wrapper.find('.flow-divination-form').classes()).toContain('flow-divination-form--loading')

    await form.trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('proxies restore and typed context through the canonical shared form', async () => {
    const wrapper = mount(FlowDivinationForm)
    const restored: DivinationCastingFormValue = {
      question: '搬家日期怎么看？',
      castingMode: 'auto',
      location: { ...savedLocation! },
    }

    wrapper.vm.restore(restored)
    await wrapper.vm.$nextTick()

    expect(wrapper.vm.getContext()).toEqual(restored)
    expect((wrapper.find('textarea[name="divination-question"]').element as HTMLTextAreaElement).value)
      .toBe(restored.question)
  })

  it('keeps one primary action and native field labelling', async () => {
    const wrapper = mount(FlowDivinationForm)
    const question = wrapper.find('textarea[name="divination-question"]')
    await nextTick()
    const label = wrapper.find('.el-form-item__label')
    const labelFor = label.attributes('for')

    expect(wrapper.findAll('form')).toHaveLength(1)
    expect(wrapper.findAll('button[type="submit"]')).toHaveLength(1)
    expect(label.text()).toBe('你想问的一件事')
    expect(labelFor).toBeTruthy()
    expect(question.attributes('id')).toBe(labelFor)
    expect(wrapper.find('.flow-divination-form').attributes('aria-busy')).toBe('false')
    expect(wrapper.find('.flow-divination-form').attributes('inert')).toBeUndefined()
  })
})
