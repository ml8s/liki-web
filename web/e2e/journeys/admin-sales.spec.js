// @ts-check
// 销售后台（订单 / 订阅）列表：表格数据、状态标签、服务端分页元数据，以及缺少
// sales.read 时的能力边界。分页契约（扁平 page/page_size/total）此前与客户端
// 读取方式不一致，导致页脚永远停在第 1 页——这里守住修复后的行为。
import { test, expect } from '../fixtures.js'

const adminUser = { id: 'user-admin', email: 'admin@example.com', status: 'active', locale: 'zh-Hans' }

const order = (id, status) => ({
  id, user_id: 'user-1', email: 'customer@example.com', product_id: 'liki-chat',
  amount_minor: 1000, currency: 'USD', status, payment_intent_id: `pi_${id}`,
  created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z', paid_at: null,
})

const subscription = (id, status) => ({
  id, user_id: 'user-1', email: 'customer@example.com', product_id: 'liki-chat',
  billing_mode: 'subscription', provider: 'dodo', provider_subscription_id: `dodo_${id}`,
  status, currency_code: 'USD', current_period_start: '2026-10-01T00:00:00Z',
  current_period_end: '2026-11-01T00:00:00Z', next_billing_at: '2026-11-01T00:00:00Z',
  cancel_at_period_end: true, cancelled_at: null,
  created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
})

async function mockAdmin(page, capabilities) {
  await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
  await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
  await page.route('**/api/admin/session', route => route.fulfill({
    json: { data: { email: adminUser.email, capabilities } },
  }))
}

test('orders list renders rows and follows server pagination', async ({ page }) => {
  const pages = []
  await mockAdmin(page, ['sales.read'])
  await page.route('**/api/admin/sales/orders*', route => {
    const current = Number(new URL(route.request().url()).searchParams.get('page') ?? 1)
    pages.push(current)
    return route.fulfill({ json: { data: {
      orders: [order(`order_${current}_a`, 'paid'), order(`order_${current}_b`, 'pending')],
      page: current, page_size: 50, total: 120, has_more: current < 3,
    } } })
  })

  await page.goto('/app/admin/orders')
  await expect(page.getByRole('heading', { name: '订单', level: 1 })).toBeVisible()
  await expect(page.getByText('order_1_a', { exact: true })).toBeVisible()
  // 状态文案来自共享的 adminStatusMeta（paid 取打款语义「已打款」）；此处只断言
  // 渲染结果，术语是否统一是另一个问题。
  await expect(page.locator('.el-table').getByText('已打款', { exact: true })).toBeVisible()

  const pager = page.locator('.admin-pagination')
  await expect(pager).toBeVisible()
  await pager.locator('li:has-text("2")').click()

  await expect(page.getByText('order_2_a', { exact: true })).toBeVisible()
  await expect(page.getByText('order_1_a', { exact: true })).toBeHidden()
  expect(pages).toEqual([1, 2])
})

test('subscription list shows provider ids and the lifecycle statuses', async ({ page }) => {
  await mockAdmin(page, ['sales.read'])
  await page.route('**/api/admin/sales/subscriptions*', route => route.fulfill({ json: { data: {
    subscriptions: [subscription('sub_hold', 'on_hold'), subscription('sub_gone', 'cancelled')],
    page: 1, page_size: 50, total: 2, has_more: false,
  } } }))

  await page.goto('/app/admin/subscriptions')
  await expect(page.getByRole('heading', { name: '订阅', level: 1 })).toBeVisible()

  const table = page.locator('.el-table')
  await expect(table.getByText('sub_hold', { exact: true })).toBeVisible()
  await expect(table.getByText('dodo_sub_hold', { exact: true })).toBeVisible()
  await expect(table.getByText('暂停')).toBeVisible()
  await expect(table.getByText('已取消')).toBeVisible()
  // 期末取消列：行内只呈现布尔语义，不暴露服务商字段。
  await expect(table.locator('.el-table__row')).toHaveCount(2)
})

test('an admin without sales.read sees no sales navigation and no data', async ({ page }) => {
  await mockAdmin(page, ['entitlement.read'])
  await page.route('**/api/admin/sales/subscriptions*', route => route.fulfill({
    status: 403,
    contentType: 'application/problem+json',
    body: JSON.stringify({ type: 'forbidden', title: 'forbidden', status: 403, detail: '无权访问' }),
  }))

  await page.goto('/app/admin')
  await expect(page.getByRole('heading', { name: '管理总览' })).toBeVisible()
  await expect(page.getByRole('link', { name: '订单' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '订阅' })).toHaveCount(0)

  await page.goto('/app/admin/subscriptions')
  await expect(page.locator('.query-error')).toBeVisible()
  await expect(page.getByText('sub_hold', { exact: true })).toHaveCount(0)
})
