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
  intent: 'natal',
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

  it('restores an active session when the list cache is empty', async () => {
    const chat = useChatStore()
    const chatApi = await import('@/api/chat')
    const durableContext = {
      version: 1,
      intent: 'hepan',
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
      return sendResult
    })

    await chat.createSession('', 'natal')
    chat.activeSession = session
    await chat.send(' 你好 ')

    expect(createSpy).toHaveBeenCalledWith('', 'liki', 'natal', '命理解读', undefined)
    // BFF 负责权威上下文；前端只传当前会话与消息。
    expect(sendSpy).toHaveBeenCalledWith(session, '你好', expect.any(Function), expect.any(AbortSignal))
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
      intent: 'divination',
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
      intent: 'hepan',
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
      'hepan',
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
      return sendResult
    })

    await chat.createSession('', 'natal')
    chat.activeSession = session
    await chat.send('你好')

    // phase 事件只带原始工具名，标签由前端 i18n 渲染。
    expect(phaseAfterTool).toEqual([i18n.global.t('chat.tool.chart')])
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
        session,
        active_skill: session.skill,
        user_message: userMessage,
        assistant_message: assistantMessage,
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
