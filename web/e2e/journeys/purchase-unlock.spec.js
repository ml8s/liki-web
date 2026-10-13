// @ts-check
// 付费解锁的 UI 表现：未购买 → 开通引导可见；已购买 → 引导消失、Chat 可用。
// 后端支付全链由 payment_runtime_test.go（Go 单测）覆盖；此处保护前端解锁逻辑。
import { test, expect } from '@playwright/test'

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const user = { id: 'usr_unlock', email: 'unlock@example.com', status: 'active', locale: 'zh-Hant' }

const baseProduct = {
  id: 'liki-chat', name: 'Liki Chat', description: 'Unified metaphysics chat.',
  skills: ['liki'], prices: { CNY: 100 }, display_currency: 'CNY', display_amount_minor: 100,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 100 } },
  ],
}

async function mockWorkspace(page, { purchased, productPurchased }) {
  // Fallback first (Playwright routes are LIFO): any API not explicitly
  // mocked below gets an empty success instead of a real 401, which would
  // trigger a hard redirectToLogin navigation away from the workspace.
  await page.route('**/api/**', route => route.fulfill(json({})))
  await page.route('**/api/me', route => route.fulfill(json(user)))
  await page.route('**/api/me/entitlements', route => route.fulfill(json({
    entitlements: purchased ? [{ id: 'ent_1', skill: 'liki', status: 'active' }] : [],
    allowed_skills: purchased ? ['liki'] : [],
  })))
  await page.route('**/api/chat/products', route => route.fulfill(json({
    products: [{ ...baseProduct, purchased: productPurchased }],
  })))
  await page.route('**/api/chat/sessions', route => route.fulfill(json([])))
  await page.route('**/api/chat/orders', route => route.fulfill(json({ orders: [] })))
  await page.route('**/api/analytics/pageview', route => route.fulfill(json({})))
}

test('locked state shows the purchase entry', async ({ page }) => {
  await mockWorkspace(page, { purchased: false, productPurchased: false })
  await page.goto('/app/chat')
  await expect(page.locator('#chat-purchase')).toBeVisible()
  await expect(page.locator('#chat-purchase')).toContainText('單次訪問')
})

test('purchased state hides the purchase entry', async ({ page }) => {
  await mockWorkspace(page, { purchased: true, productPurchased: true })
  await page.goto('/app/chat')
  await expect(page.locator('#chat-purchase')).toBeHidden()
  // Core workspace still present (locale-independent structural markers).
  await expect(page.locator('.profile-picker')).toBeVisible()
  await expect(page.locator('.skill-grid')).toBeVisible()
})
