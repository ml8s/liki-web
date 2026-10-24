import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import type { ChatMessage, ChatSession } from '@/api/chat'
import { i18n } from '@/i18n'

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
const userMessage = {
  id: 'msg_1', session_id: session.id, user_id: session.user_id,
  role: 'user', kind: 'text', content: '你好', created_at: '2026-01-01T00:01:00Z',
} satisfies ChatMessage
const assistantMessage = {
  ...userMessage, id: 'msg_2', role: 'assistant', content: '你好，请提供信息。',
} satisfies ChatMessage

beforeEach(() => {
  setActivePinia(createPinia())
  vi.restoreAllMocks()
  global.fetch = vi.fn()
})

describe('chat store', () => {
  it('rejects an unlicensed skill before calling create API', async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { entitlements: [], allowed_skills: [] } }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ))
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()

    await expect(chat.createSession('', 'natal')).rejects.toThrow('当前订单未开通该 skill')
    expect(entitlements.hasSkill('liki')).toBe(false)
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })

  it('restores a direct session link when the list cache is empty', async () => {
    const chat = useChatStore()
    const chatApi = await import('@/api/chat')
    const durableContext = {
      version: 1,
      scenario: 'hepan',
      second_birth: {
        gender: 'female',
        date: '1990-05-12',
        time: '08:30',
        location: { longitude: 112.9823, timezone: 'Asia/Shanghai' },
      },
      third_party_consent: true,
    } as const
    const detailedSession = { ...session, context: durableContext }
    const sessionSpy = vi.spyOn(chatApi, 'fetchChatSession').mockResolvedValue(detailedSession)
    const messagesSpy = vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue({ messages: [userMessage, assistantMessage], next_cursor: '' })

    await chat.openSession(session.id)

    expect(sessionSpy).toHaveBeenCalledWith(session.id)
    expect(messagesSpy).toHaveBeenCalledWith(session.id)
    expect(chat.activeSession?.id).toBe(session.id)
    expect(chat.activeSession?.context).toEqual(durableContext)
    expect(chat.messages).toHaveLength(2)
  })

  it('persists both returned messages and activates the session', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'relay')
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    const createSpy = vi.spyOn(chatApi, 'createChatSession').mockResolvedValue(session)
    const sendResult = {
      session,
      active_skill: session.skill,
      user_message: userMessage,
      assistant_message: assistantMessage,
    }
    const sendSpy = vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_sessionId, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: assistantMessage.content })
      return { transport: 'relay', result: sendResult } as const
    })

    await chat.createSession('', 'natal')
    chat.activeSession = session
    await chat.send(' 你好 ')

    expect(createSpy).toHaveBeenCalledWith('', 'liki', 'natal', '命理解读', undefined)
    // 中继模式不携带直连上下文（第 5 参为 undefined）。
    expect(sendSpy).toHaveBeenCalledWith(session, '你好', expect.any(Function), expect.any(AbortSignal), undefined)
    expect(chat.messages).toEqual([userMessage, assistantMessage])
    expect(chat.activeSession?.id).toBe(session.id)
  })

  it('sends durable consultation context when creating a tool-originated session', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true
    const chatApi = await import('@/api/chat')
    const createSpy = vi.spyOn(chatApi, 'createChatSession').mockResolvedValue(session)
    const context = {
      version: 1,
      scenario: 'divination',
      question: '這個項目能成嗎',
    } as const

    await chat.createSession('profile-1', 'divination', context)

    expect(createSpy).toHaveBeenCalledWith(
      'profile-1',
      'liki',
      'divination',
      '问卦解读',
      context,
    )
  })

  it('always creates a new session when a tool supplies consultation context', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true
    chat.sessions = [{ ...session, profile_id: 'profile-1' }]

    const chatApi = await import('@/api/chat')
    const createSpy = vi.spyOn(chatApi, 'createChatSession').mockResolvedValue(session)
    const context = {
      version: 1,
      scenario: 'hepan',
      second_birth: {
        gender: 'female',
        date: '1990-05-12',
        time: '08:30',
        location: { longitude: 112.9823, timezone: 'Asia/Shanghai' },
      },
      third_party_consent: true,
    } as const

    await chat.ensureSession('profile-1', 'hepan', context)

    expect(createSpy).toHaveBeenCalledWith(
      'profile-1',
      'liki',
      'natal',
      expect.any(String),
      context,
    )
  })

  it('localizes tool phases from raw tool names', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'createChatSession').mockResolvedValue(session)
    const sendResult = {
      session,
      active_skill: session.skill,
      user_message: userMessage,
      assistant_message: assistantMessage,
    }
    const phaseAfterTool: string[] = []
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_sessionId, _content, onEvent) => {
      onEvent({ type: 'phase', content: 'bazi_chart' })
      phaseAfterTool.push(chat.phase)
      onEvent({ type: 'text-delta', content: assistantMessage.content })
      return { transport: 'relay', result: sendResult } as const
    })

    await chat.createSession('', 'natal')
    chat.activeSession = session
    await chat.send('你好')

    // phase 事件只带原始工具名，标签由前端 i18n 渲染。
    expect(phaseAfterTool).toEqual([i18n.global.t('chat.tool.chart')])
  })
})

  it('keeps the local exchange on the direct transport (history comes from the callback)', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'createChatSession').mockResolvedValue(session)
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_session, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: '直连答案' })
      return { transport: 'direct', answer: '直连答案' } as const
    })

    await chat.createSession('', 'natal')
    chat.activeSession = session
    await chat.send('你好')

    expect(chat.messages).toHaveLength(2)
    expect(chat.messages[1].content).toBe('直连答案')
    expect(chat.error).toBe('')
  })

describe('direct transport context', () => {
  it('carries conversation history and the profile context on the direct transport', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'direct')
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const consultationContext = {
      version: 1,
      scenario: 'divination',
      question: '項目能否成交',
      casting_instant: '2026-10-23T04:00:00Z',
    } as const
    const profileSession = { ...session, profile_id: 'prof_1', context: consultationContext }
    const chatApi = await import('@/api/chat')
    const birthApi = await import('@/api/birth-profiles')
    const profileSpy = vi.spyOn(birthApi, 'fetchBirthProfile').mockResolvedValue({
      id: 'prof_1',
      user_id: 'usr_1',
      version: 1,
      nickname: '测试',
      name: '',
      birth: {
        gender: 'male',
        date: '1990-05-20',
        time: '11:00',
        instant: '1990-05-20T03:00:00Z',
        location: { place: 'Beijing', longitude: 116.4074, timezone: 'Asia/Shanghai' },
      },
      note: '',
      third_party_consent: false,
      last_used_at: null,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })
    const sendSpy = vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_session, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: '答案' })
      return { transport: 'direct', answer: '答案' } as const
    })

    chat.activeSession = profileSession
    chat.messages = [userMessage, assistantMessage]
    await chat.send('继续')

    expect(profileSpy).toHaveBeenCalledWith('prof_1')
    expect(sendSpy).toHaveBeenCalledWith(profileSession, '继续', expect.any(Function), expect.any(AbortSignal), {
      history: [
        { id: userMessage.id, role: 'user', content: userMessage.content },
        { id: assistantMessage.id, role: 'assistant', content: assistantMessage.content },
      ],
      profile: {
        gender: 'male',
        instant: '1990-05-20T03:00:00Z',
        location: { longitude: 116.4074, timezone: 'Asia/Shanghai' },
      },
      consultationContext,
    })
    vi.unstubAllEnvs()
  })

  it('fails closed when a profile-bound direct run cannot load context', async () => {
    vi.stubEnv('VITE_CHAT_TRANSPORT', 'direct')
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const profileSession = { ...session, profile_id: 'prof_1' }
    const chatApi = await import('@/api/chat')
    const birthApi = await import('@/api/birth-profiles')
    vi.spyOn(birthApi, 'fetchBirthProfile').mockRejectedValue(new Error('boom'))
    const sendSpy = vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_session, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: '答案' })
      return { transport: 'direct', answer: '答案' } as const
    })

    chat.activeSession = profileSession
    await chat.send('你好')

    expect(sendSpy).not.toHaveBeenCalled()
    expect(chat.error).toBeTruthy()
    expect(chat.sending).toBe(false)
    vi.unstubAllEnvs()
  })
})

describe('chat streaming failure states', () => {
  it('reloads authoritative messages after a failed stream', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_sessionId, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: '半个' })
      const { ApiError } = await import('@/api/http')
      throw new ApiError(500, 'generation_failed', 'Generation failed')
    })
    const history = [userMessage]
    vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue({ messages: history, next_cursor: '' })

    chat.activeSession = session
    await chat.send('你好')

    expect(chat.error).toBe(i18n.global.t('errors.code.generation_failed'))
    expect(chat.messages).toEqual(history)
    expect(chat.sending).toBe(false)
  })

  it('keeps the partial answer and marks it stopped after stopping generation', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_sessionId, _content, onEvent, signal) => {
      onEvent({ type: 'text-delta', content: '部分回复' })
      await Promise.resolve()
      signal?.throwIfAborted()
      return {
        transport: 'relay',
        result: {
          session,
          active_skill: session.skill,
          user_message: userMessage,
          assistant_message: assistantMessage,
        },
      } as const
    })
    vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue({ messages: [userMessage], next_cursor: '' })

    chat.activeSession = session
    const sending = chat.send('你好')
    chat.stopGeneration()
    await sending

    // 停止生成：保留本地已流出的部分内容并标记 stopped，不做 refetch（部分答案不得静默消失）。
    expect(chat.error).toBe('')
    expect(chat.messages).toHaveLength(2)
    expect(chat.messages[0]).toMatchObject({ role: 'user', content: '你好' })
    expect(chat.messages[1]).toMatchObject({ role: 'assistant', content: '部分回复', stopped: true })
    expect(chat.sending).toBe(false)
  })
})
