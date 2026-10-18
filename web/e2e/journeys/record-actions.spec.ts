// 记录列表行操作的标准回路：会话（重命名/删除）与订单（继续支付/取消/去使用/过期态）。
// 两个列表都走 RecordActions（.record-action），断言同时保护 API 请求与 UI 状态迁移。
import { test, expect } from '../fixtures'
import type { Page } from '@playwright/test'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })

const user = testUser({ id: 'usr_actions', email: 'actions@example.com' })

const sessions = [
  {
    id: 'chat_1', user_id: user.id, profile_id: 'p_1', skill: 'liki', expert: 'natal', title: '八字諮詢',
    status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z', last_message_at: null,
  },
  {
    id: 'chat_2', user_id: user.id, profile_id: 'p_1', skill: 'liki', expert: 'natal', title: '流年運勢',
    status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', last_message_at: null,
  },
]

const order = (id: string, status: string, intentStatus: string) => ({
  id,
  user_id: user.id,
  email: user.email,
  product_id: 'liki-chat',
  amount_minor: 7900,
  currency: 'CNY',
  status,
  payment_intent_id: `pi_${id}`,
  intent_status: intentStatus,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  paid_at: status === 'paid' ? '2026-01-01T00:05:00Z' : null,
})

async function mockOrdersPage(page: Page, orders: unknown[]) {
  await mockWorkspace(page, {
    user,
    allowedSkills: ['liki'],
    sessions: [],
    orders,
    entitlements: [{
      id: 'ent_1', user_id: user.id, product_id: 'liki-chat', skills: ['liki'], status: 'active',
      starts_at: '2026-01-01T00:00:00Z', expires_at: '2027-01-01T00:00:00Z',
    }],
  })
  await page.route('**/api/sales/subscriptions', route => route.fulfill(json([])))
  await page.route('**/api/sales/subscription-payments', route => route.fulfill(json([])))
}

test('聊天会话行操作：重命名与删除走标准请求', async ({ page }) => {
  await mockWorkspace(page, { user, allowedSkills: ['liki'], sessions })
  const patchBodies: unknown[] = []
  let deleted = 0
  await page.route('**/api/chat/sessions/chat_1', route => {
    patchBodies.push(route.request().postDataJSON())
    const body = route.request().postDataJSON() as { title: string }
    return route.fulfill(json({ ...sessions[0], title: body.title }))
  })
  await page.route('**/api/chat/sessions/chat_2', route => {
    deleted += 1
    return route.fulfill(json({ status: 'deleted' }))
  })

  await page.goto('/app/chat')
  await expect(page.locator('.el-table__row')).toHaveCount(2)

  const firstRow = page.locator('.el-table__row', { hasText: '八字諮詢' })
  await firstRow.getByRole('button', { name: '重新命名' }).click()
  const prompt = page.locator('.el-message-box')
  await expect(prompt).toBeVisible()
  await prompt.locator('.el-message-box__input input').fill('新的主題')
  await prompt.getByRole('button', { name: '儲存' }).click()
  await expect(page.locator('.el-table__row', { hasText: '新的主題' })).toBeVisible()
  expect(patchBodies).toEqual([{ title: '新的主題' }])

  const secondRow = page.locator('.el-table__row', { hasText: '流年運勢' })
  await secondRow.getByRole('button', { name: '刪除' }).click()
  const confirm = page.locator('.el-message-box')
  await expect(confirm).toContainText('流年運勢')
  await confirm.getByRole('button', { name: '刪除' }).click()
  await expect(page.locator('.el-table__row')).toHaveCount(1)
  expect(deleted).toBe(1)
})

test('订单行操作：继续支付跳收银台、取消生效为已取消', async ({ page }) => {
  const orders = [
    order('ord_resume', 'pending', 'checkout_created'),
    order('ord_cancel', 'pending', 'checkout_created'),
    order('ord_expired', 'pending', 'expired'),
    order('ord_paid', 'paid', 'paid'),
  ]
  await mockOrdersPage(page, orders)
  await page.route('**/api/sales/orders/ord_paid', route => route.fulfill(json({
    order: orders[3],
    payment: {
      provider: 'xunhu',
      status: 'paid',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:05:00Z',
      paid_at: '2026-01-01T00:05:00Z',
      refunded_at: null,
      refunded_amount_minor: 0,
    },
  })))
  const checkouts: string[] = []
  const cancels: string[] = []
  await page.route('**/api/sales/orders/ord_resume/checkout', route => {
    checkouts.push(route.request().url())
    return route.fulfill(json({
      order_id: 'ord_resume',
      payment_intent_id: 'pi_ord_resume',
      provider: 'xunhu',
      status: 'checkout_created',
      checkout_url: '/pay/stub',
      status_token: 'token',
    }))
  })
  await page.route('**/api/sales/orders/ord_cancel/cancel', route => {
    cancels.push(route.request().url())
    return route.fulfill(json({ status: 'canceled' }))
  })
  await page.route('**/pay/stub', route => route.fulfill({ body: '<html><body>收银台占位</body></html>', contentType: 'text/html' }))

  await page.goto('/app/orders')
  await expect(page.locator('.orders-table')).toBeVisible()

  // 未付可续付/取消；过期不可续付、只可重新购买；已付可直接使用。
  const resumeRow = page.locator('.el-table__row', { hasText: 'ord_resume' })
  await expect(resumeRow.getByRole('button', { name: '繼續支付' })).toBeVisible()
  await expect(resumeRow.getByRole('button', { name: '取消訂單' })).toBeVisible()
  const expiredRow = page.locator('.el-table__row', { hasText: 'ord_expired' })
  await expect(expiredRow).toContainText('已過期')
  await expect(expiredRow.getByRole('link', { name: '重新購買' })).toHaveAttribute('href', '/app/chat')
  // 过期订单仍可取消整理列表（分析初衷：pending 不堆积）。
  await expect(expiredRow.getByRole('button', { name: '取消訂單' })).toBeVisible()
  const paidRow = page.locator('.el-table__row', { hasText: 'ord_paid' })
  await expect(paidRow.getByRole('link', { name: '去使用' })).toHaveAttribute('href', '/app/chat')

  // 详情抽屉：支付渠道/时间线/权益生效信息。
  await paidRow.getByRole('button', { name: '詳情' }).click()
  const drawer = page.locator('.el-drawer')
  await expect(drawer).toBeVisible()
  await expect(drawer).toContainText('虎皮椒')
  await expect(drawer).toContainText('支付完成')
  await expect(drawer).toContainText('生效至')
  await page.keyboard.press('Escape')
  await expect(drawer).toBeHidden()

  await resumeRow.getByRole('button', { name: '繼續支付' }).click()
  await page.waitForURL('**/pay/stub')
  expect(checkouts).toHaveLength(1)

  await page.goto('/app/orders')
  const cancelRow = page.locator('.el-table__row', { hasText: 'ord_cancel' })
  await cancelRow.getByRole('button', { name: '取消訂單' }).click()
  const confirm = page.locator('.el-message-box')
  await expect(confirm).toBeVisible()
  await confirm.getByRole('button', { name: '取消訂單' }).click()
  await expect(cancelRow).toContainText('已取消')
  expect(cancels).toHaveLength(1)
})
