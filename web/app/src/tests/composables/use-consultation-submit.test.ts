import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import type { ChatSession } from '@/api/chat'
import type { ConsultationContext } from '@shared/domain/consultation'

const { useChatStore } = await import('@/stores/chat')
const { useConsultationSubmit } = await import('@/composables/useConsultationSubmit')

function session(overrides: Partial<ChatSession>): ChatSession {
  return {
    id: 's1',
    user_id: 'usr_1',
    profile_id: 'p1',
    skill: 'liki',
    intent: 'natal',
    title: '本命',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_message_at: null,
    ...overrides,
  }
}

const namingContext: ConsultationContext = {
  version: 1,
  intent: 'naming',
  birth: {
    gender: 'female',
    date: '1990-05-12',
    time: '08:30',
    location: { longitude: 121.47, timezone: 'Asia/Shanghai' },
  },
}

describe('useConsultationSubmit', () => {
  function setup(personId: string | null = 'p1') {
    let submit!: ReturnType<typeof useConsultationSubmit>
    mount(defineComponent({
      setup() {
        submit = useConsultationSubmit({
          personId: () => personId,
          onError: () => {},
          onDraftPreserved: () => {},
          onDraftRestored: () => {},
        })
        return () => null
      },
    }))
    return submit
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('reuses a session whose durable context matches exactly', async () => {
    const chat = useChatStore()
    chat.sessions = [session({
      id: 'existing',
      context: namingContext,
    })]
    const openSpy = vi.spyOn(chat, 'openSession').mockImplementation(async id => {
      chat.activeSession = chat.sessions.find(s => s.id === id) ?? null
    })
    const sendSpy = vi.spyOn(chat, 'send').mockResolvedValue()
    const createSpy = vi.spyOn(chat, 'createSession')

    const submit = setup()
    await submit.submitFocusedForm({
      skill: 'naming',
      context: namingContext,
      matchesContext: s => s.context?.intent === 'naming',
      content: '方向如何？',
    })

    expect(openSpy).toHaveBeenCalledWith('existing')
    expect(sendSpy).toHaveBeenCalledWith('方向如何？')
    expect(createSpy).not.toHaveBeenCalled()
  })

  it('never matches sessions from a different person', async () => {
    const chat = useChatStore()
    chat.sessions = [session({
      id: 'other-person',
      profile_id: 'p2',
      context: namingContext,
    })]
    const openSpy = vi.spyOn(chat, 'openSession').mockImplementation(async id => {
      chat.activeSession = chat.sessions.find(s => s.id === id) ?? null
    })
    const sendSpy = vi.spyOn(chat, 'send').mockResolvedValue()
    const createSpy = vi.spyOn(chat, 'createSession').mockResolvedValue(
      session({ id: 'created' }),
    )

    const submit = setup()
    await submit.submitFocusedForm({
      skill: 'naming',
      context: namingContext,
      matchesContext: s => s.context?.intent === 'naming',
      content: '方向如何？',
    })

    expect(openSpy).not.toHaveBeenCalledWith('other-person')
    expect(createSpy).toHaveBeenCalledOnce()
    expect(sendSpy).toHaveBeenCalledOnce()
  })

  it('clears the draft before send and restores it on failure', async () => {
    const flow = (await import('@/stores/domain-flow')).useDomainFlowStore()
    const chat = useChatStore()
    chat.sessions = [session({ id: 'existing', context: namingContext })]
    vi.spyOn(chat, 'openSession').mockImplementation(async id => {
      chat.activeSession = chat.sessions.find(s => s.id === id) ?? null
    })
    vi.spyOn(chat, 'send').mockImplementation(async () => {
      chat.error = '暂時無法連線服務'
    })
    flow.setDraft('方向如何？')

    const submit = setup()
    await submit.run(async () => {
      await submit.submitFocusedForm({
        skill: 'naming',
        context: namingContext,
        matchesContext: s => s.context?.intent === 'naming',
        content: '方向如何？',
      })
    })

    expect(flow.getDraft()).toBe('方向如何？')
    expect(chat.error).toBeTruthy()
  })

  it('guards concurrent submissions', async () => {
    const chat = useChatStore()
    chat.sessions = [session({ id: 'existing', context: namingContext })]
    const sendSpy = vi.spyOn(chat, 'send').mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 20)),
    )
    vi.spyOn(chat, 'openSession').mockImplementation(async id => {
      chat.activeSession = chat.sessions.find(s => s.id === id) ?? null
    })

    const submit = setup()
    const first = submit.run(async () => {
      await submit.submitFocusedForm({
        skill: 'naming',
        context: namingContext,
        matchesContext: s => s.context?.intent === 'naming',
        content: 'first',
      })
    })
    const second = submit.run(async () => {
      await submit.submitFocusedForm({
        skill: 'naming',
        context: namingContext,
        matchesContext: s => s.context?.intent === 'naming',
        content: 'second',
      })
    })
    await Promise.all([first, second])

    expect(sendSpy).toHaveBeenCalledOnce()
  })
})
