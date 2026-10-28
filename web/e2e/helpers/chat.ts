import type { Page } from '@playwright/test'

export type JsonRecord = Record<string, unknown>

interface AGUIMessage {
  role?: string
  content?: unknown
}

export interface BFFChatRequest {
  url: string
  body: JsonRecord
}

export interface BFFChatMockOptions {
  session: JsonRecord
  answer: string
  failuresBeforeSuccess?: number
  permanentFailure?: boolean
  errorMessage?: string
}

function sessionID(session: JsonRecord): string {
  const id = session.id
  if (typeof id !== 'string' || !id) {
    throw new Error('BFF chat mock requires a non-empty session.id')
  }
  return id
}

function message(
  session: JsonRecord,
  id: string,
  role: 'user' | 'assistant',
  content: string,
): JsonRecord {
  return {
    id,
    session_id: sessionID(session),
    user_id: session.user_id,
    role,
    kind: 'text',
    content,
    created_at: '2026-01-01T00:01:00Z',
  }
}

export function bffChatStream(options: {
  session: JsonRecord
  answer: string
  question?: string
  runId?: string
}): string {
  const runId = options.runId ?? 'run_bff'
  const threadId = sessionID(options.session)
  const messageId = `${runId}:assistant`
  const userMessage = message(options.session, `${runId}:user`, 'user', options.question ?? 'e2e question')
  const assistantMessage = message(options.session, messageId, 'assistant', options.answer)
  const events = [
    { type: 'RUN_STARTED', threadId, runId },
    { type: 'TEXT_MESSAGE_START', messageId, role: 'assistant' },
    { type: 'TEXT_MESSAGE_CONTENT', messageId, delta: options.answer },
    { type: 'TEXT_MESSAGE_END', messageId },
    {
      type: 'RUN_FINISHED',
      threadId,
      runId,
      outcome: { type: 'success' },
      result: {
        session: options.session,
        active_skill: options.session.skill,
        user_message: userMessage,
        assistant_message: assistantMessage,
      },
    },
  ]
  return events.map(event => `data: ${JSON.stringify(event)}\n\n`).join('')
}

function errorStream(session: JsonRecord, runId: string, errorMessage: string): string {
  return `data: ${JSON.stringify({ type: 'RUN_ERROR', runId, message: errorMessage })}\n\n`
}

function latestUserContent(body: JsonRecord): string {
  const messages = Array.isArray(body.messages) ? body.messages as AGUIMessage[] : []
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const item = messages[index]
    if (!item) continue
    if (item.role === 'user' && typeof item.content === 'string' && item.content.trim()) {
      return item.content
    }
  }
  return 'e2e question'
}

export async function mockBFFChatMessage(
  page: Page,
  options: BFFChatMockOptions,
): Promise<BFFChatRequest[]> {
  const requests: BFFChatRequest[] = []
  const runId = 'run_bff'
  await page.route(`**/api/chat/sessions/${sessionID(options.session)}/ag-ui`, async route => {
    const body = route.request().postDataJSON() as JsonRecord
    requests.push({ url: route.request().url(), body })
    if (options.permanentFailure || requests.length <= (options.failuresBeforeSuccess ?? 0)) {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: errorStream(options.session, runId, options.errorMessage ?? 'engine unavailable'),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: bffChatStream({
        session: options.session,
        answer: options.answer,
        question: latestUserContent(body),
        runId,
      }),
    })
  })
  return requests
}
