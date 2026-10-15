// 支付返回（/app/chat?order_id=..&status=..）在浏览器的收尾：把一次性回调参数从
// 地址栏消费掉，并且只按后端确认的状态说话。轮询/退避分支由
// web/app/src/tests/sales-payment-return.test.ts 用假计时器覆盖。
import { test, expect } from '../fixtures'

const json = (data: unknown) => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const user = { id: 'usr_pay', email: 'pay@example.com', status: 'active', locale: 'zh-Hant' }

test('an unfinished payment stays on the list and is reported as such', async ({ page }) => {
  let sessionCreates = 0
  // 兜底先注册（Playwright 路由后注册优先）：未显式 mock 的 API 返回空成功，
  // 避免真实 401 触发整页跳登录。
  await page.route('**/api/**', route => route.fulfill(json({})))
  await page.route('**/api/me', route => route.fulfill(json(user)))
  await page.route('**/api/me/entitlements', route => route.fulfill(json({ entitlements: [], allowed_skills: [] })))
  await page.route('**/api/sales/products', route => route.fulfill(json({ products: [] })))
  await page.route('**/api/chat/sessions', route => {
    if (route.request().method() === 'POST') sessionCreates += 1
    return route.fulfill(json([]))
  })
  await page.route('**/api/birth-profiles**', route => route.fulfill(json({ profiles: [], next_cursor: '' })))

  await page.goto('/app/chat?order_id=order_1&status=unpaid')

  await expect(page.getByText('支付尚未完成')).toBeVisible()
  await expect(page).toHaveURL(/\/app\/chat$/)
  expect(sessionCreates).toBe(0)
})
