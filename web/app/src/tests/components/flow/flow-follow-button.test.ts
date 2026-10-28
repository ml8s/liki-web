import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import FlowFollowButton from '@/components/flow/FlowFollowButton.vue'

describe('FlowFollowButton', () => {
  it('renders its label and emits follow', async () => {
    const wrapper = mount(FlowFollowButton, {
      props: { label: '有新内容' },
    })

    expect(wrapper.get('button').text()).toBe('有新内容')
    await wrapper.trigger('click')
    expect(wrapper.emitted('follow')).toHaveLength(1)
  })
})
