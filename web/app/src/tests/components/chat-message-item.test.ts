import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ChatMessageItem from '@/components/chat/ChatMessageItem.vue'
import type { ChatMessage } from '@/api/chat'

describe('ChatMessageItem', () => {
  it('renders user message and data role', () => {
    const message: ChatMessage = {
      id: 'msg_1', session_id: 'session_1', user_id: 'usr_1',
      role: 'user', kind: 'text', content: '你好', created_at: '2026-01-01T00:00:00Z',
    }
    const wrapper = mount(ChatMessageItem, { props: { message } })
    expect(wrapper.text()).toContain('我')
    expect(wrapper.text()).toContain('你好')
    expect(wrapper.find('[data-role="user"]').exists()).toBe(true)
  })
})
