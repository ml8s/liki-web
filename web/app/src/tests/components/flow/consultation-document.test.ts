import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ConsultationDocument from '@/components/flow/ConsultationDocument.vue'
import type { ChatMessage } from '@/api/chat'

function message(overrides: Partial<ChatMessage>): ChatMessage {
  return {
    id: 'message-1',
    session_id: 'session-1',
    user_id: 'user-1',
    role: 'assistant',
    kind: 'text',
    content: '',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('ConsultationDocument', () => {
  it('renders questions and assistant readings as a document', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [
          message({
            id: 'q1',
            role: 'user',
            content: '今年事业怎么样？',
          }),
          message({ id: 'a1', content: '# 结论\n\n事业节奏稳定。' }),
        ],
      },
    })

    expect(wrapper.findAll('.turn')).toHaveLength(2)
    expect(wrapper.find('.question__text').text()).toBe('今年事业怎么样？')
    expect(wrapper.find('.reading .app-markdown').html()).toContain('结论')
  })

  it('does not expose system profile records as consultation turns', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [
          message({ id: 'system', role: 'assistant', kind: 'system_profile', content: 'profile' }),
          message({ id: 'a1', content: '结论' }),
        ],
      },
    })

    expect(wrapper.findAll('.turn')).toHaveLength(1)
  })

  it('shows a loading skeleton for an empty streaming assistant turn', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [message({ content: '' })],
        streaming: true,
      },
    })

    expect(wrapper.find('.reading__loading [aria-busy="true"]').exists()).toBe(true)
  })

  it('shows an initial loading state without inventing consultation content', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [],
        loading: true,
      },
    })

    expect(wrapper.get('.consultation-document').attributes('aria-busy')).toBe('true')
    expect(wrapper.find('.document-loading [aria-busy="true"]').exists()).toBe(true)
    expect(wrapper.find('.app-markdown').exists()).toBe(false)
  })

  it('renders a local generation error without leaking it into markdown', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [message({ error: '生成失败' })],
      },
    })

    expect(wrapper.find('.reading__error').text()).toBe('生成失败')
    expect(wrapper.find('.reading__error').attributes('role')).toBe('alert')
    expect(wrapper.find('.app-markdown').exists()).toBe(false)
  })

  it('marks an active stream as busy while retaining the semantic document structure', () => {
    const wrapper = mount(ConsultationDocument, {
      props: {
        messages: [message({ id: 'q1', role: 'user', content: '问题' })],
        streaming: true,
      },
    })

    expect(wrapper.get('.consultation-document').attributes('aria-busy')).toBe('true')
    expect(wrapper.get('.consultation-document').element.tagName).toBe('OL')
    expect(wrapper.findAll('.turn')).toHaveLength(1)
  })
})
