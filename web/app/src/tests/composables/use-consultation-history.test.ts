import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import type { ChatSession } from '@/api/chat'

const promptRenameEntity = vi.hoisted(() => vi.fn())
const confirmDeleteEntity = vi.hoisted(() => vi.fn())

vi.mock('@/utils/dialogs', () => ({
  promptRenameEntity,
  confirmDeleteEntity,
}))

const { useChatStore } = await import('@/stores/chat')
const { useConsultationHistory } = await import('@/composables/useConsultationHistory')

function session(overrides: Partial<ChatSession>): ChatSession {
  return {
    id: 's1',
    user_id: 'usr_1',
    profile_id: 'p1',
    skill: 'liki',
    expert: 'natal',
    title: '本命',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_message_at: null,
    ...overrides,
  }
}

describe('useConsultationHistory', () => {
  function withHistory(personId: () => string | null, domainId: () => string) {
    let history!: ReturnType<typeof useConsultationHistory>
    mount(defineComponent({
      setup() {
        history = useConsultationHistory({ personId, domainId })
        return () => null
      },
    }))
    return history
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    promptRenameEntity.mockReset()
    confirmDeleteEntity.mockReset()
  })

  it('filters sessions by the visible person and domain, newest first', () => {
    const chat = useChatStore()
    chat.sessions = [
      session({ id: 'old', updated_at: '2026-01-01T00:00:00Z' }),
      session({ id: 'new', updated_at: '2026-01-02T00:00:00Z' }),
      session({ id: 'other-person', profile_id: 'p2' }),
      session({ id: 'other-domain', expert: 'divination' }),
    ]

    const history = withHistory(() => 'p1', () => 'natal')

    expect(history.visibleSessions.value.map(s => s.id)).toEqual(['new', 'old'])
    expect(history.latestVisibleSession.value?.id).toBe('new')
  })

  it('hides restore when the latest session is already active', () => {
    const chat = useChatStore()
    chat.sessions = [session({ id: 'latest' })]
    chat.activeSession = session({ id: 'latest' })

    const history = withHistory(() => 'p1', () => 'natal')
    expect(history.restoreVisible.value).toBe(false)
  })

  it('resetActive when restoring with no visible session', async () => {
    const chat = useChatStore()
    chat.sessions = []
    const resetSpy = vi.spyOn(chat, 'resetActive')

    const history = withHistory(() => 'p1', () => 'natal')
    await history.restoreLatest()

    expect(resetSpy).toHaveBeenCalledOnce()
  })

  it('restoreLatest opens the newest visible session', async () => {
    const chat = useChatStore()
    chat.sessions = [session({ id: 'latest', updated_at: '2026-01-02T00:00:00Z' })]
    const openSpy = vi.spyOn(chat, 'openSession').mockResolvedValue()

    const history = withHistory(() => 'p1', () => 'natal')
    await history.restoreLatest()

    expect(openSpy).toHaveBeenCalledWith('latest')
  })

  it('surfaces rename failures on the store without throwing', async () => {
    const chat = useChatStore()
    const target = session({ id: 's1' })
    promptRenameEntity.mockResolvedValue('新标题')
    vi.spyOn(chat, 'renameSession').mockRejectedValue(new Error('boom'))

    const history = withHistory(() => 'p1', () => 'natal')
    await history.renameSession(target)

    expect(chat.error).toBeTruthy()
  })
})
