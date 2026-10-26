import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import FlowNamingForm from '@/components/flow/forms/FlowNamingForm.vue'
import FormShell from '@shared/components/form/FormShell.vue'

describe('FlowNamingForm', () => {
  it('shows the selected birth profile and disables incomplete facts', () => {
    const wrapper = mount(FlowNamingForm, {
      props: {
        personLabel: '张三',
        birthDate: '',
        birthTime: '',
        disabled: true,
      },
    })

    expect(wrapper.text()).toContain('张三')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('submits the current profile when facts are complete', async () => {
    const wrapper = mount(FlowNamingForm, {
      props: {
        personLabel: '张三',
        birthDate: '1990-06-15',
        birthTime: '08:30',
      },
    })

    wrapper.getComponent(FormShell).vm.$emit('submit')
    expect(wrapper.emitted('submit')).toHaveLength(1)
    wrapper.unmount()
  })

  it('does not bypass disabled through FormShell submit', () => {
    const wrapper = mount(FlowNamingForm, {
      props: {
        personLabel: '张三',
        birthDate: '1990-06-15',
        birthTime: '08:30',
        disabled: true,
      },
    })
    wrapper.getComponent(FormShell).vm.$emit('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    wrapper.unmount()
  })
})
