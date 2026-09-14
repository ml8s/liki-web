import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ChatComposer from '@/components/chat/ChatComposer.vue'

describe('ChatComposer', () => {
  it('does not submit blank content', async () => {
    const wrapper = mount(ChatComposer, { props: { modelValue: '   ', sending: false } })
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('submits non-empty content', async () => {
    const wrapper = mount(ChatComposer, { props: { modelValue: '你好', sending: false } })
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('shows stop instead of submit while sending', () => {
    const wrapper = mount(ChatComposer, { props: { modelValue: '你好', sending: true } })
    expect(wrapper.text()).toContain('停止')
  })
})
