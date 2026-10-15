// @ts-check
// 会话页的两条体验底线：流式生成失败必须给出可重试的明确反馈（并能真的重发成功）；
// 窄屏下必须能读能发、不横向溢出。这两处在单测里无法覆盖。
import { test, expect } from '@playwright/test'

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const sse = events => events.map(event => `data: ${JSON.stringify(event)}\n\n`).join('')
const stream = body => ({ status: 200, contentType: 'text/event-stream', body })

const user = { id: 'usr_chat', email: 'chat@example.com', status: 'active', locale: 'zh-Hant' }
const session = {
  id: 'chat_1', user_id: user.id, profile_id: 'p_1', skill: 'liki', title: '八字諮詢',
  status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  last_message_at: null,
}
const message = (id, role, content) => ({
  id, session_id: session.id, user_id: user.id, role, kind: 'text', content,
  created_at: '2026-01-01T00:01:00Z',
})
const RUN_ID = 'run_1'
const answer = '先看你的八字，日主偏弱。'
const successStream = sse([
  { type: 'RUN_STARTED', threadId: session.id, runId: RUN_ID },
  { type: 'TEXT_MESSAGE_START', messageId: `${RUN_ID}:assistant`, role: 'assistant' },
  { type: 'TEXT_MESSAGE_CONTENT', messageId: `${RUN_ID}:assistant`, delta: answer },
  { type: 'TEXT_MESSAGE_END', messageId: `${RUN_ID}:assistant` },
  {
    type: 'RUN_FINISHED',
    threadId: session.id,
    runId: RUN_ID,
    outcome: { type: 'success' },
    result: {
      session,
      active_skill: 'liki',
      user_message: message('msg_user', 'user', answer),
      assistant_message: message('msg_assistant', 'assistant', answer),
    },
  },
])

async function mockSession(page, messages) {
  await page.route('**/api/**', route => route.fulfill(json({})))
  await page.route('**/api/me', route => route.fulfill(json(user)))
  await page.route('**/api/me/entitlements', route => route.fulfill(json({ entitlements: [], allowed_skills: ['liki'] })))
  await page.route('**/api/chat/sessions/chat_1', route => route.fulfill(json(session)))
  await page.route('**/api/chat/sessions/chat_1/messages', route => route.fulfill(json(messages)))
}

test('a failed stream offers retry and resending succeeds', async ({ page }) => {
  const sends = []
  await mockSession(page, [])
  await page.route('**/api/chat/sessions/chat_1/ag-ui', route => {
    sends.push(route.request().postDataJSON())
    if (sends.length === 1) {
      return route.fulfill(stream(sse([{ type: 'RUN_ERROR', message: 'engine unavailable', runId: RUN_ID }])))
    }
    return route.fulfill(stream(successStream))
  })

  await page.goto('/app/chat/chat_1')
  await expect(page.getByRole('heading', { name: '八字諮詢' })).toBeVisible()

  const composer = page.locator('[name="chat-message"]')
  await composer.fill('我的八字如何？')
  await page.getByRole('button', { name: '發送' }).click()

  // 失败：给出可读原因并可重试，草稿保留（按钮可直接重发原消息）。
  const errorRow = page.locator('.error-row')
  await expect(errorRow).toContainText('生成失敗')
  const retry = page.getByRole('button', { name: '重試' })
  await expect(retry).toBeVisible()

  const thread = page.getByRole('log', { name: '訊息列表' })
  await retry.click()
  await expect(thread).toContainText(answer)
  await expect(errorRow).toBeHidden()
  expect(sends).toHaveLength(2)
})

test('the conversation owns the viewport and follows the keyboard', async ({ page }) => {
  // 会话页独占视口（route meta viewportBound）：文档不滚动，滚动发生在消息区，
  // 输入框常驻可见；键盘弹起时（visualViewport 变矮）消息区跟到最新一条。
  const history = Array.from({ length: 40 }, (_, index) => message(`msg_${index}`, 'user', `第 ${index + 1} 則訊息`))
  await mockSession(page, history)
  await page.route('**/api/chat/sessions/chat_1/ag-ui', route => route.fulfill(stream(successStream)))

  await page.goto('/app/chat/chat_1')

  // 外层 .thread 才是滚动容器（内层 ChatThread 用同名 class 做 aria-live 列表）。
  const thread = page.locator('.chat-main > .thread')
  const distanceToEnd = () => thread.evaluate(el => el.scrollHeight - el.scrollTop - el.clientHeight)
  const composerFits = () => page.evaluate(() => {
    const box = document.querySelector('form.composer')?.getBoundingClientRect()
    return !!box && box.top >= 0 && box.bottom <= window.innerHeight + 0.5
  })

  await expect.poll(distanceToEnd).toBeLessThan(40)
  expect(await thread.evaluate(el => el.scrollHeight > el.clientHeight)).toBe(true)
  expect(await composerFits()).toBe(true)
  expect(await page.evaluate(() => document.scrollingElement.scrollHeight - window.innerHeight)).toBeLessThanOrEqual(1)

  await page.setViewportSize({ width: 1280, height: 500 })
  await page.evaluate(() => window.visualViewport?.dispatchEvent(new Event('resize')))

  await expect.poll(distanceToEnd).toBeLessThan(40)
  expect(await composerFits()).toBe(true)
})

test('mobile: a message can be sent in a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockSession(page, [message('msg_1', 'assistant', '先給我出生資訊。')])
  await page.route('**/api/chat/sessions/chat_1/ag-ui', route => route.fulfill(stream(successStream)))

  await page.goto('/app/chat/chat_1')
  const thread = page.getByRole('log', { name: '訊息列表' })
  await expect(thread).toContainText('先給我出生資訊。')

  // 窄屏最容易坏的是横向溢出与输入：两者都必须保持可用。
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)

  await page.locator('[name="chat-message"]').fill('在手機上也要能問。')
  await page.getByRole('button', { name: '發送' }).click()
  await expect(thread).toContainText(answer)
})
