import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import FlowHepanForm from '@/components/flow/forms/FlowHepanForm.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'

const validBirth = {
  gender: 'female' as const,
  date: '1992-06-18',
  time: '10:30',
  location: {
    place: '',
    longitude: 116.4,
    timezone: 'Asia/Shanghai',
    source: 'saved' as const,
  },
}

describe('FlowHepanForm', () => {
  it('does not submit while second-person consent or birth facts are missing', async () => {
    const wrapper = mount(FlowHepanForm, {
      props: { personLabel: '张三' },
    })

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toBeUndefined()
    wrapper.unmount()
  })

  it('validates the other person and emits affirmative consent once', async () => {
    const wrapper = mount(FlowHepanForm, {
      props: { personLabel: '张三' },
    })
    wrapper.findComponent(BirthFields).vm.$emit('update:patch', validBirth)
    await wrapper.vm.$nextTick()
    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.emitted('submit')?.[0]).toEqual([{
      secondBirth: validBirth,
      thirdPartyConsent: true,
    }])
    wrapper.unmount()
  })
})
