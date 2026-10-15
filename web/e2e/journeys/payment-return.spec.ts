// 支付返回（/app/chat?order_id=..&status=..）在浏览器的收尾：把一次性回调参数从
// 地址栏消费掉，并且只按后端确认的状态说话。轮询/退避分支由
// web/app/src/tests/sales-payment-return.test.ts 用假计时器覆盖。
import { test, expect } from '../fixtures'
import { json, mockWorkspace, testUser } from '../mocks'

const user = testUser({ id: 'usr_pay', email: 'pay@example.com' })

test('an unfinished payment stays on the list and is reported as such', async ({ page }) => {
  let sessionCreates = 0
  await mockWorkspace(page, { user })
  // 覆盖壳里的会话列表：未完成的支付不应该建会话（路由后注册优先）。
  await page.route('**/api/chat/sessions', route => {
    if (route.request().method() === 'POST') sessionCreates += 1
    return route.fulfill(json([]))
  })

  await page.goto('/app/chat?order_id=order_1&status=unpaid')

  await expect(page.getByText('支付尚未完成')).toBeVisible()
  await expect(page).toHaveURL(/\/app\/chat$/)
  expect(sessionCreates).toBe(0)
})
