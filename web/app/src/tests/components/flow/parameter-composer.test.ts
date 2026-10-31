import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ParameterComposer from '@/components/flow/ParameterComposer.vue'
import { appDomains } from '@shared/domain/flow'
import type { AppDomainId } from '@shared/domain/flow'

const baseProps = {
  domains: appDomains,
  domainId: 'natal' as AppDomainId,
  personLabel: '张三',
  domainLabel: '命理',
}

async function pressEnter(
  wrapper: ReturnType<typeof mount<typeof ParameterComposer>>,
  init: KeyboardEventInit,
) {
  const textarea = wrapper.find('textarea')
  const event = new KeyboardEvent('keydown', { bubbles: true, cancelable: true, ...init })
  textarea.element.dispatchEvent(event)
  await wrapper.vm.$nextTick()
  return event
}

describe('ParameterComposer', () => {
  it('submits non-empty questions and rejects blank input without mutating it', async () => {
    const wrapper = mount(ParameterComposer, {
      props: { ...baseProps, modelValue: '   ' },
    })

    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await wrapper.setProps({ modelValue: '今年事业怎么样？' })
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('uses Enter to submit, Shift+Enter for a newline, and ignores IME Enter', async () => {
    const wrapper = mount(ParameterComposer, {
      props: { ...baseProps, modelValue: '流年怎么看？' },
    })

    const composingEvent = await pressEnter(wrapper, { key: 'Enter', isComposing: true })
    expect(composingEvent.defaultPrevented).toBe(false)
    expect(wrapper.emitted('submit')).toBeUndefined()

    const shiftEvent = await pressEnter(wrapper, { key: 'Enter', shiftKey: true })
    expect(shiftEvent.defaultPrevented).toBe(false)
    expect(wrapper.emitted('submit')).toBeUndefined()

    const submitEvent = await pressEnter(wrapper, { key: 'Enter' })
    expect(submitEvent.defaultPrevented).toBe(true)
    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('replaces submission with stop while sending', async () => {
    const wrapper = mount(ParameterComposer, {
      props: { ...baseProps, modelValue: '进行中的问题', sending: true },
    })

    expect(wrapper.find('textarea').attributes('disabled')).toBeDefined()
    expect(wrapper.find('button[type="submit"]').exists()).toBe(false)

    await wrapper.find('.flow-send').trigger('click')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('stop')).toHaveLength(1)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('exposes the person control', async () => {
    const wrapper = mount(ParameterComposer, {
      props: { ...baseProps, modelValue: '', showDomainStrip: true },
    })
    const parameters = wrapper.findAll('.flow-parameter')

    expect(parameters).toHaveLength(1)
    expect(parameters[0]?.text()).toBe('张三')
    expect(parameters[0]?.attributes('aria-haspopup')).toBe('dialog')

    await parameters[0]?.trigger('click')
    expect(wrapper.emitted('select-person')).toHaveLength(1)
  })

  it('keeps the question as controlled consultation text rather than routing input', async () => {
    const onUpdate = vi.fn()
    const wrapper = mount(ParameterComposer, {
      props: {
        ...baseProps,
        modelValue: '帮我选一个开业日期',
        showDomainStrip: true,
        'onUpdate:modelValue': onUpdate,
      },
    })

    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.emitted('update:domainId')).toBeUndefined()
    expect(onUpdate).not.toHaveBeenCalled()
  })
})
