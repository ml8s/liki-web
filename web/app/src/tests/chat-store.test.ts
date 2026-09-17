import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import type { ChatMessage, ChatSession } from '@/api/chat'

const session = {
  id: 'chat_1',
  user_id: 'usr_1',
  profile_id: '',
  skill: 'liki',
  title: '命理咨询',
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

    await expect(chat.createSession('', 'liki')).rejects.toThrow('当前订单未开通该 skill')
    expect(entitlements.hasSkill('liki')).toBe(false)
    expect(global.fetch).toHaveBeenCalledTimes(1)
  })

  it('restores a direct session link when the list cache is empty', async () => {
    const chat = useChatStore()
    const chatApi = await import('@/api/chat')
    const sessionSpy = vi.spyOn(chatApi, 'fetchChatSession').mockResolvedValue(session)
    const messagesSpy = vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue([userMessage, assistantMessage])

    await chat.openSession(session.id)

    expect(sessionSpy).toHaveBeenCalledWith(session.id)
    expect(messagesSpy).toHaveBeenCalledWith(session.id)
    expect(chat.activeSession?.id).toBe(session.id)
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
      onEvent({ type: 'thinking' })
      onEvent({ type: 'text-delta', content: assistantMessage.content })
      return sendResult
    })

    await chat.createSession('', 'liki')
    chat.activeSession = session
    await chat.send(' 你好 ')

    expect(createSpy).toHaveBeenCalledWith('', 'liki', '命理咨询')
    expect(sendSpy).toHaveBeenCalledWith(session.id, '你好', expect.any(Function), expect.any(AbortSignal))
    expect(chat.messages).toEqual([userMessage, assistantMessage])
    expect(chat.activeSession?.id).toBe(session.id)
  })
})

describe('chat streaming failure states', () => {
  it('reloads authoritative messages after an SSE error event', async () => {
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    entitlements.allowedSkills = ['liki']
    entitlements.loaded = true

    const chatApi = await import('@/api/chat')
    vi.spyOn(chatApi, 'streamChatMessage').mockImplementation(async (_sessionId, _content, onEvent) => {
      onEvent({ type: 'text-delta', content: '半个' })
      onEvent({ type: 'error', content: '生成失败' })
      throw new Error('生成失败')
    })
    const history = [userMessage]
    vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue(history)

    chat.activeSession = session
    await chat.send('你好')

    expect(chat.error).toBe('生成失败')
    expect(chat.messages).toEqual(history)
    expect(chat.sending).toBe(false)
  })

  it('reloads authoritative messages after stopping generation', async () => {
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
      }
    })
    vi.spyOn(chatApi, 'fetchChatMessages').mockResolvedValue([userMessage])

    chat.activeSession = session
    const sending = chat.send('你好')
    chat.stopGeneration()
    await sending

    expect(chat.error).toBe('')
    expect(chat.messages).toEqual([userMessage])
    expect(chat.sending).toBe(false)
  })
})
