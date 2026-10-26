import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import FlowFengshuiForm from '@/components/flow/forms/FlowFengshuiForm.vue'
import FormShell from '@shared/components/form/FormShell.vue'

async function selectGua(wrapper: ReturnType<typeof mount>, index: number, gua: string) {
  const select = wrapper.findAllComponents({ name: 'ElSelect' })[index]
  select?.vm.$emit('update:modelValue', gua)
  await wrapper.vm.$nextTick()
}

describe('FlowFengshuiForm', () => {
  it('disables submit until person facts and all layout guas are complete', async () => {
    const wrapper = mount(FlowFengshuiForm, {
      props: {
        personLabel: '张三',
        birthYear: 1990,
        gender: 'female',
      },
    })
    const submit = wrapper.find('button[type="submit"]')
    expect(submit.attributes('disabled')).toBeDefined()

    await selectGua(wrapper, 1, '乾')
    await selectGua(wrapper, 2, '坤')
    await selectGua(wrapper, 3, '兑')
    await wrapper.vm.$nextTick()

    expect(submit.attributes('disabled')).toBeUndefined()
    wrapper.getComponent(FormShell).vm.$emit('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({
      zuoShan: 0,
      doorGua: '乾',
      masterGua: '坤',
      stoveGua: '兑',
    })
    wrapper.unmount()
  })

  it('disables submit when the selected profile cannot derive a valid birth year', () => {
    const wrapper = mount(FlowFengshuiForm, {
      props: {
        personLabel: '张三',
        birthYear: null,
        gender: '' as const,
      },
    })

    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
