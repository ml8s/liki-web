import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import FlowHistoryPanel from '@/components/flow/FlowHistoryPanel.vue'
import type { ChatSession } from '@/api/chat'

function session(id: string, title: string): ChatSession {
  return {
    id,
    user_id: 'user-1',
    profile_id: 'person-1',
    skill: 'liki',
    expert: 'natal',
    title,
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
    last_message_at: null,
  }
}

const sessions = [
  session('latest', '流年事业解读'),
  session('older', '风水客厅布局'),
]

describe('FlowHistoryPanel', () => {
  it('filters scoped sessions by title and marks the active session', async () => {
    const wrapper = mount(FlowHistoryPanel, {
      props: {
        sessions,
        activeSessionId: 'latest',
      },
    })

    expect(wrapper.text()).toContain('2 条会话')
    expect(wrapper.text()).toContain('流年事业解读')
    expect(wrapper.get('.flow-history__item.is-active').text()).toContain('当前')

    await wrapper.find('input[type="search"]').setValue('风水')
    await wrapper.vm.$nextTick()

    const titles = wrapper.findAll('.flow-history__title').map(node => node.text())
    expect(titles).toEqual(['风水客厅布局'])
    wrapper.unmount()
  })

  it('emits scoped session actions', async () => {
    const wrapper = mount(FlowHistoryPanel, {
      props: { sessions },
    })

    await wrapper.get('.flow-history__open').trigger('click')
    const actions = wrapper.findAll('.flow-history__action')
    await actions[0]!.trigger('click')
    await actions[1]!.trigger('click')

    expect(wrapper.emitted('open')?.[0]).toEqual([sessions[0]])
    expect(wrapper.emitted('rename')?.[0]).toEqual([sessions[0]])
    expect(wrapper.emitted('remove')?.[0]).toEqual([sessions[0]])
    wrapper.unmount()
  })
})
