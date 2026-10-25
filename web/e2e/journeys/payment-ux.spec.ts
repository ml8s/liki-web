// 支付返回与订单页反馈：返回订单必须明确提示，订阅取消必须可见，桌面微信必须提示。
import { test, expect } from '../fixtures'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })

const order = {
  id: 'order_1', product_id: 'liki-chat', amount_minor: 7900, currency: 'CNY',
  status: 'pending', payment_intent_id: 'pi_1', intent_status: 'checkout_created',
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', paid_at: null,
}

const cnyProduct = {
  id: 'liki-chat', name: 'Liki Chat', description: 'Liki Chat.', skills: ['liki'],
  display_currency: 'CNY', display_amount_minor: 7900,
  purchase_options: [{ billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 7900 } }],
  purchased: false,
}

async function mockOrders(page: Parameters<typeof mockWorkspace>[0], orders: unknown[] = []) {
  await mockWorkspace(page, { user: testUser(), orders })
  await page.route('**/api/sales/subscriptions', route => route.fulfill(json([])))
  await page.route('**/api/sales/subscription-payments', route => route.fulfill(json([])))
  await page.route('**/api/me/sessions', route => route.fulfill(json([])))
  await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))
}

test('returning to an unpaid order shows the status and highlights it', async ({ page }) => {
  await mockOrders(page, [order])
  await page.goto('/app/orders?order_id=order_1&status=unpaid')

  await expect(page.getByText('支付未完成', { exact: true })).toBeVisible()
  await expect(page.locator('tr.order-row-focus')).toHaveCount(1)
  await expect(page.locator('tr.order-row-focus')).toContainText('order_1')
})

test('a cancelled subscription checkout is reported on chat', async ({ page }) => {
  await mockWorkspace(page, { user: testUser(), allowedSkills: [] })
  await page.route('**/api/me/sessions', route => route.fulfill(json([])))
  await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))

  await page.goto('/app/chat?subscription_state=cancelled')
  await expect(page.getByText(/訂閱支付已取消|订阅支付已取消|Subscription checkout was cancelled/)).toBeVisible()
})

test('CNY checkout shows the desktop WeChat hint', async ({ page }) => {
  await mockWorkspace(page, { user: testUser(), allowedSkills: [], products: [cnyProduct] })
  await page.route('**/api/me/sessions', route => route.fulfill(json([])))
  await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))

  await page.goto('/app/chat')
  await expect(page.getByText(/桌面端會跳轉到微信支付收銀台|桌面端会跳转到微信支付收银台|On desktop you will be redirected/)).toBeVisible()
})
