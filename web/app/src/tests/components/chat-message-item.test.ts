import { describe, expect, it, vi } from 'vitest'
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

  it('renders assistant markdown safely', async () => {
    const message: ChatMessage = {
      id: 'msg_2', session_id: 'session_1', user_id: 'usr_1',
      role: 'assistant', kind: 'text',
      content: '## 结论\n\n**重要**\n\n<script>alert(1)</script>',
      created_at: '2026-01-01T00:00:00Z',
    }
    const wrapper = mount(ChatMessageItem, { props: { message } })
    // AppMarkdown is lazy (O4); flush the async component before asserting.
    await vi.waitFor(() => {
      expect(wrapper.find('.app-markdown').exists()).toBe(true)
    })
    const html = wrapper.html()
    expect(wrapper.text()).toContain('AI 顾问')
    expect(wrapper.text()).toContain('结论')
    expect(wrapper.find('h2').exists()).toBe(true)
    expect(wrapper.find('strong').text()).toBe('重要')
    expect(html).not.toContain('<script>alert(1)</script>')
  })
})
