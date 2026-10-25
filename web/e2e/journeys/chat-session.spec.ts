// 会话页的两条体验底线：流式生成失败必须给出可重试的明确反馈（并能真的重发成功）；
// 窄屏下必须能读能发、不横向溢出。这两处在单测里无法覆盖。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { json, mockWorkspace, testUser } from '../mocks'

const sse = (events: unknown[]) => events.map(event => `data: ${JSON.stringify(event)}\n\n`).join('')
const stream = (body: string) => ({ status: 200, contentType: 'text/event-stream', body })

const user = testUser({ id: 'usr_chat', email: 'chat@example.com' })
const session = {
  id: 'chat_1', user_id: user.id, profile_id: 'p_1', skill: 'liki', expert: 'natal', title: '八字諮詢',
  status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  last_message_at: null,
}
const message = (id: string, role: string, content: string) => ({
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

async function mockSession(page: Page, messages: unknown[]) {
  await mockWorkspace(page, { user, allowedSkills: ['liki'] })
  await page.route(/\/api\/chat\/sessions\/chat_1$/, route => route.fulfill(json(session)))
  await page.route('**/api/chat/sessions/chat_1/messages', route =>
    route.fulfill(json({ messages, next_cursor: '' })))
}

test('更早的消息按游标分页加载', async ({ page }) => {
  const older = message('msg_older', 'user', '更早的問題')
  const recent = message('msg_recent', 'assistant', '最近的回覆')
  await mockWorkspace(page, { user, allowedSkills: ['liki'] })
  await page.route(/\/api\/chat\/sessions\/chat_1$/, route => route.fulfill(json(session)))
  await page.route(/\/api\/chat\/sessions\/chat_1\/messages(?:\?.*)?$/, route => {
    const before = new URL(route.request().url()).searchParams.get('before')
    if (before) {
      return route.fulfill(json({ messages: [older], next_cursor: '' }))
    }
    return route.fulfill(json({ messages: [recent], next_cursor: '2026-01-01T00:01:00Z|msg_recent' }))
  })

  await page.goto('/app/chat/chat_1')
  const thread = page.getByRole('log', { name: '訊息列表' })
  await expect(thread).toContainText('最近的回覆')

  // 顶部出现「載入更早」→ 点击后更早消息插入，游标用尽按钮消失。
  const earlier = page.getByRole('button', { name: '載入更早的訊息' })
  await expect(earlier).toBeVisible()
  await earlier.click()
  await expect(thread).toContainText('更早的問題')
  await expect(earlier).toBeHidden()
})

test('a failed stream offers retry and resending succeeds', async ({ page }) => {
  const sends: unknown[] = []
  await mockSession(page, [])
  await page.route(/\/api\/chat\/sessions\/chat_1\/ag-ui$/, route => {
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
  await page.route(/\/api\/chat\/sessions\/chat_1\/ag-ui$/, route => route.fulfill(stream(successStream)))

  await page.goto('/app/chat/chat_1')

  // 外层 .thread 才是滚动容器（内层 ChatThread 用同名 class 做 aria-live 列表）。
  const thread = page.locator('.chat-main > .thread')
  const distanceToEnd = () => thread.evaluate(el => el.scrollHeight - el.scrollTop - el.clientHeight)
  const composerFits = () => page.evaluate(() => {
    const box = document.querySelector('form.composer')?.getBoundingClientRect()
    return !!box && box.top >= 0 && box.bottom <= window.innerHeight + 0.5
  })

  // 先等历史渲染完，再断言滚动职责：空列表时 scrollHeight == clientHeight，
  // 提前断言会把「还没渲染」误判成「没有溢出的滚动容器」。
  await expect(thread).toContainText('第 40 則訊息')
  await expect.poll(() => thread.evaluate(el => el.scrollHeight > el.clientHeight)).toBe(true)
  await expect.poll(distanceToEnd).toBeLessThan(40)
  expect(await composerFits()).toBe(true)
  expect(await page.evaluate(() => (document.scrollingElement?.scrollHeight ?? 0) - window.innerHeight)).toBeLessThanOrEqual(1)

  await page.setViewportSize({ width: 1280, height: 500 })
  await page.evaluate(() => window.visualViewport?.dispatchEvent(new Event('resize')))

  await expect.poll(distanceToEnd).toBeLessThan(40)
  expect(await composerFits()).toBe(true)
})

test('mobile: a message can be sent in a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const history = Array.from({ length: 30 }, (_, index) => (
    message(`msg_${index + 1}`, index % 2 ? 'assistant' : 'user', `第 ${index + 1} 條訊息`)
  ))
  await mockSession(page, history)
  await page.route(/\/api\/chat\/sessions\/chat_1\/ag-ui$/, route => route.fulfill(stream(successStream)))

  await page.goto('/app/chat/chat_1')
  const thread = page.getByRole('log', { name: '訊息列表' })
  await expect(thread).toContainText('第 30 條訊息')

  const copyHeight = await page.locator('.copy').first().evaluate(el => el.getBoundingClientRect().height)
  expect(copyHeight).toBeGreaterThanOrEqual(44)

  const scrollThread = page.locator('.chat-main > .thread')
  await scrollThread.evaluate(el => {
    el.scrollTop = 0
  })
  const jump = page.locator('.jump-latest')
  await expect(jump).toBeVisible()
  const controlGeometry = await page.evaluate(() => {
    const jumpBox = document.querySelector('.jump-latest')?.getBoundingClientRect()
    const composerBox = document.querySelector('form.composer')?.getBoundingClientRect()
    return {
      jumpHeight: jumpBox?.height ?? 0,
      jumpBottom: jumpBox?.bottom ?? Number.NEGATIVE_INFINITY,
      composerTop: composerBox?.top ?? Number.POSITIVE_INFINITY,
    }
  })
  expect(controlGeometry.jumpHeight).toBeGreaterThanOrEqual(44)
  expect(controlGeometry.jumpBottom).toBeLessThan(controlGeometry.composerTop)

  // 窄屏最容易坏的是横向溢出与输入：两者都必须保持可用。
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)

  await page.locator('[name="chat-message"]').fill('在手機上也要能問。')
  await page.getByRole('button', { name: '發送' }).click()
  await expect(thread).toContainText(answer)
})
