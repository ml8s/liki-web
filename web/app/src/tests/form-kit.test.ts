import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ChoiceCardGroup from '@shared/components/form/fields/ChoiceCardGroup.vue'
import DateField from '@shared/components/form/fields/DateField.vue'
import NumberField from '@shared/components/form/fields/NumberField.vue'
import SelectField from '@shared/components/form/fields/SelectField.vue'
import TextField from '@shared/components/form/fields/TextField.vue'

describe('shared form kit', () => {
  it('wires text labels, hints, and model updates', async () => {
    const wrapper = mount(TextField, {
      props: {
        modelValue: '',
        name: 'email',
        label: 'Email',
        hint: 'Use your login email',
      },
    })
    await nextTick()

    const input = wrapper.find('input')
    const label = wrapper.find('label')
    expect(label.attributes('for')).toBe(input.attributes('id'))
    expect(input.attributes('aria-describedby')).toBe(wrapper.find('.lf-field__hint').attributes('id'))

    await input.setValue('user@example.com')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['user@example.com'])
  })

  it('marks controlled errors invalid and describes them', async () => {
    const wrapper = mount(TextField, {
      props: {
        modelValue: '',
        name: 'email',
        label: 'Email',
        error: 'Email is required',
      },
    })

    const input = wrapper.find('input')
    const error = wrapper.find('.lf-field__error')
    expect(error.text()).toBe('Email is required')
    expect(input.attributes('aria-invalid')).toBe('true')
    expect(input.attributes('aria-describedby')).toBe(error.attributes('id'))
  })

  it('emits typed and cleared number values', async () => {
    const wrapper = mount(NumberField, {
      props: {
        modelValue: null,
        name: 'year',
        label: 'Year',
      },
    })

    const inputNumber = wrapper.findComponent({ name: 'ElInputNumber' })
    inputNumber.vm.$emit('update:modelValue', 1990)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([1990])

    inputNumber.vm.$emit('update:modelValue', undefined)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
  })

  it('normalizes cleared date values to an empty string', async () => {
    const wrapper = mount(DateField, {
      props: {
        modelValue: '1990-05-12',
        name: 'birth-date',
        label: 'Birth date',
      },
    })

    // ElDatePicker is lazy (O4); wait for the async component before emitting.
    // ElDatePicker is lazy (O4); wait for the async component to mount.
    await vi.waitFor(() => {
      expect(wrapper.findComponent({ name: 'ElDatePicker' }).exists()).toBe(true)
    })
    wrapper.findComponent({ name: 'ElDatePicker' }).vm.$emit('update:modelValue', null)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([''])
  })

  it('normalizes cleared select values to null', () => {
    const wrapper = mount(SelectField, {
      props: {
        modelValue: 'female',
        name: 'gender',
        label: 'Gender',
        options: [
          { value: 'female', label: 'Female' },
          { value: 'male', label: 'Male' },
        ],
      },
    })

    wrapper.findComponent({ name: 'ElSelect' }).vm.$emit('update:modelValue', undefined)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
  })

  it('emits selected choice-card values and shows one selected state', async () => {
    const wrapper = mount(ChoiceCardGroup, {
      props: {
        modelValue: 'female',
        name: 'gender',
        label: 'Gender',
        options: [
          { value: 'female', label: 'Female' },
          { value: 'male', label: 'Male' },
        ],
      },
    })

    wrapper.findComponent({ name: 'ElRadioGroup' }).vm.$emit('update:modelValue', 'male')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['male'])
    await wrapper.setProps({ modelValue: 'male' })
    expect(wrapper.findAll('.choice-card__state--selected')).toHaveLength(1)
  })
})
