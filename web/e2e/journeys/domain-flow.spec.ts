import { expect, test } from '../fixtures'
import {
  flowMessage,
  flowProfile,
  flowSession,
  mockFlowWorkspace,
  openFlowDomainSheet,
  openFlowPersonSheet,
  type JsonRecord,
} from '../helpers/domain-flow'

const profiles = [
  flowProfile('person-1', '张三'),
  flowProfile('person-2', '李四'),
]

async function awaitFlowReady(page: Parameters<typeof mockFlowWorkspace>[0]) {
  await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
}

function sseAnswer(answer: string) {
  const runId = 'run_flow'
  const messageId = `${runId}:assistant`
  return [
    { type: 'RUN_STARTED', threadId: 'chat_new', runId },
    { type: 'TEXT_MESSAGE_START', messageId, role: 'assistant' },
    { type: 'TEXT_MESSAGE_CONTENT', messageId, delta: answer },
    { type: 'TEXT_MESSAGE_END', messageId },
    {
      type: 'RUN_FINISHED',
      threadId: 'chat_new',
      runId,
      outcome: { type: 'success' },
      result: { answer },
    },
  ].map(event => `data: ${JSON.stringify(event)}\n\n`).join('')
}

test.describe('four-domain consultation flow', () => {
  test.describe('mobile', () => {
    test.use({ viewport: { width: 375, height: 720 } })

    test('exposes exactly the four canonical domains and no legacy task level', async ({ page }) => {
      await mockFlowWorkspace(page, { profiles })
      await page.goto('/app/consultation')
      await awaitFlowReady(page)
      await expect(page.getByRole('tab', { name: '命理' })).toBeVisible()

      const tabs = page.locator('.flow-domain-strip [role="tab"]')
      await expect(tabs).toHaveText(['命理', '问卦', '风水', '起名'])

      const sheet = await openFlowDomainSheet(page)
      const options = sheet.locator('.domain-list label span:not(.domain-check)')
      await expect(options).toHaveText(['命理', '问卦', '风水', '起名'])
      await expect(sheet.getByRole('button', { name: '开始新咨询' })).toBeVisible()

      await page.locator('.flow-sheet__close').click()
      await expect(page.locator('.flow-sheet__body')).toBeHidden()

      const classicSwitcher = page.locator('.experience-switcher--dock')
      await expect(classicSwitcher).toHaveText('经典版')
      await classicSwitcher.click()
      await expect(page).toHaveURL(/\/app\/?$/)
    })

    test('isolates question drafts by person and domain', async ({ page }) => {
      const session = flowSession('chat_natal', { updated_at: '2026-01-02T00:00:00Z' })
      await mockFlowWorkspace(page, { profiles, sessions: [session] })
      await page.route('**/api/chat/sessions/chat_natal', route =>
        route.fulfill({ json: { data: session } }))
      await page.route('**/api/chat/sessions/chat_natal/messages', route =>
        route.fulfill({ json: { data: { messages: [], next_cursor: '' } } }))

      await page.goto('/app/consultation')
      await awaitFlowReady(page)
      const question = page.getByRole('textbox', { name: '直接提出你的问题' })
      await question.fill('张三的命理问题')

      const personSheet = await openFlowPersonSheet(page, '张三')
      await personSheet.locator('label', { hasText: '李四' }).click()
      await expect(question).toHaveValue('')

      await question.fill('李四的命理问题')
      await openFlowPersonSheet(page, '李四')
      await page.locator('.flow-sheet__body').locator('label', { hasText: '张三' }).click()

      await expect(question).toHaveValue('张三的命理问题')
      await expect(page.locator('.flow-domain-strip [role="tab"]', { hasText: '命理' })).toHaveAttribute('aria-selected', 'true')
    })
  })

  test('restores the latest visible natal session after a new consultation', async ({ page }) => {
    const session = flowSession('chat_latest', { updated_at: '2026-01-03T00:00:00Z' })
    await mockFlowWorkspace(page, {
      profiles: [profiles[0]!],
      sessions: [
        flowSession('chat_older', { updated_at: '2026-01-01T00:00:00Z' }),
        session,
      ],
    })
    await page.route('**/api/chat/sessions/chat_latest', route =>
      route.fulfill({ json: { data: session } }))
    await page.route('**/api/chat/sessions/chat_latest/messages', route =>
      route.fulfill({ json: { data: {
        messages: [
          flowMessage('question-1', 'chat_latest', 'user', '今年事业如何？'),
          flowMessage('answer-1', 'chat_latest', 'assistant', '先看你的八字，日主偏弱。'),
        ],
        next_cursor: '',
      } } }))

    await page.goto('/app/consultation')
    await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('先看你的八字，日主偏弱。')

    const sheet = await openFlowDomainSheet(page)
    await sheet.getByRole('button', { name: '开始新咨询' }).click()
    await expect(page.locator('.flow-restore')).toBeVisible()
    await page.locator('.flow-restore').click()

    await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('今年事业如何？')
    await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('先看你的八字，日主偏弱。')
  })

  test('creates a free-form natal question on the natal expert path', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles: [profiles[0]!] })

    const sessionCreations: JsonRecord[] = []
    await page.route('**/api/chat/sessions', async route => {
      if (route.request().method() !== 'POST') {
        return route.fulfill({ json: { data: [] } })
      }
      sessionCreations.push(route.request().postDataJSON() as JsonRecord)
      return route.fulfill({ json: { data: flowSession('chat_new', {
        updated_at: '2026-01-04T00:00:00Z',
      }) } })
    })
    await page.route('**/api/birth-profiles/person-1', route =>
      route.fulfill({ json: { data: profiles[0] } }))

    const runtimeRequests: string[] = []
    await page.route(/\/rt\/natal\/ag-ui$/, async route => {
      runtimeRequests.push(route.request().url())
      return route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: sseAnswer('先看你的八字，日主偏弱。'),
      })
    })

    await page.goto('/app/consultation')
    await awaitFlowReady(page)
    await page.getByRole('textbox', { name: '直接提出你的问题' }).fill('今年事业如何？')
    await page.getByRole('button', { name: '发送' }).click()

    await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('先看你的八字，日主偏弱。')
    expect(sessionCreations).toHaveLength(1)
    expect(sessionCreations[0]).toMatchObject({
      profile_id: 'person-1',
      skill: 'liki',
      expert: 'natal',
      scenario: 'natal',
    })
    expect(runtimeRequests).toHaveLength(1)
  })

  test('keeps an explicit huangli seed pending instead of routing to divination', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles: [profiles[0]!] })

    let sessionCreationCount = 0
    let runtimeRequestCount = 0
    await page.route('**/api/chat/sessions', async route => {
      if (route.request().method() !== 'POST') {
        return route.fulfill({ json: { data: [] } })
      }
      sessionCreationCount += 1
      return route.fulfill({ status: 500, json: { data: null } })
    })
    await page.route(/\/rt\/(natal|divination)\/ag-ui$/, async route => {
      runtimeRequestCount += 1
      return route.abort('failed')
    })

    await page.goto('/app/consultation?seed=huangli')
    await awaitFlowReady(page)
    const question = page.getByRole('textbox', { name: '直接提出你的问题' })
    await question.fill('下个月哪天适合签约？')
    await page.getByRole('button', { name: '发送' }).click()

    await expect(page.locator('.flow-error')).toContainText('正在迁移到命理专家')
    await expect(question).toHaveValue('下个月哪天适合签约？')
    expect(sessionCreationCount).toBe(0)
    expect(runtimeRequestCount).toBe(0)
  })
})
