import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import ChatThread from '@/components/chat/ChatThread.vue'
import { useChatStore } from '@/stores/chat'
import type { ChatSession } from '@/api/chat'

const session = {
  id: 'chat_1',
  user_id: 'usr_1',
  profile_id: '',
  skill: 'liki',
  expert: 'natal',
  title: '本命解读',
  status: 'active',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  last_message_at: null,
} satisfies ChatSession

describe('ChatThread direct rendering', () => {
  // 回归：直连完成态必须触发 ChatThread 重渲染。
  // 历史 bug：assistant 写的是 push 前的原始对象，绕过了 Vue 代理，DOM 永久为空。
  it('renders the direct answer once send() resolves', async () => {
    setActivePinia(createPinia())
    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_session, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: 'E2E_FULL_CAPABILITY_ANSWER' })
      return { transport: 'direct', answer: 'E2E_FULL_CAPABILITY_ANSWER' }
    })

    const chat = useChatStore()
    chat.activeSession = session

    const wrapper = mount(ChatThread, { props: { messages: chat.messages } })
    await chat.send('你好')
    await nextTick()

    expect(wrapper.text()).toContain('E2E_FULL_CAPABILITY_ANSWER')
  })
})
