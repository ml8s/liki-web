import { test, expect } from '../fixtures.js'

const adminUser = { id: 'user-admin', email: 'admin@example.com', status: 'active', locale: 'zh-Hans' }

test.describe('Vue admin console', () => {
  test('renders capability-scoped workspace and manages entitlement search journey', async ({ page }) => {
    await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
    await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
    await page.route('**/api/admin/session', route => route.fulfill({
      json: { data: { email: adminUser.email, capabilities: ['entitlement.read', 'entitlement.manage'] } },
    }))
    await page.route('**/api/admin/entitlements*', route => route.fulfill({
      json: { data: { entitlements: [{
        id: 'ent_1', user_id: 'user-1', email: 'customer@example.com', product_id: 'liki-chat',
        skills: ['liki'], status: 'active', source: 'payment', source_id: 'order_1',
        starts_at: '2026-10-01T00:00:00Z', expires_at: '2026-11-01T00:00:00Z',
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
      }] } },
    }))

    await page.goto('/app/admin')
    await expect(page).toHaveURL(/\/app\/admin$/)
    await expect(page.getByRole('heading', { name: '管理总览' })).toBeVisible()
    await expect(page.locator('.operator').getByText('admin@example.com')).toBeVisible()
    await expect(page.getByText('权益管理')).toBeVisible()
    await expect(page.getByRole('link', { name: '提现' })).toHaveCount(0)

    await page.getByRole('link', { name: '权益' }).click()
    await page.locator('.filters').getByPlaceholder('user@example.com').fill('customer@example.com')
    await page.getByRole('button', { name: '查询' }).click()
    await expect(page.getByText('customer@example.com').first()).toBeVisible()
    await expect(page.getByText('payment', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: '改期' })).toBeVisible()
    await expect(page.getByRole('button', { name: '撤销' })).toBeVisible()
  })

  test('shows the no-permission state for a signed-in non-admin', async ({ page }) => {
    await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
    await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
    await page.route('**/api/admin/session', route => route.fulfill({
      json: { data: { email: adminUser.email, capabilities: [] } },
    }))

    await page.goto('/app/admin')
    await expect(page.getByText('无管理权限')).toBeVisible()
    await expect(page.getByRole('button', { name: '切换账号' })).toBeVisible()
  })
})

test.describe('Vue admin finance and compensation', () => {
  test('shows period finance totals', async ({ page }) => {
    await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
    await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
    await page.route('**/api/admin/session', route => route.fulfill({
      json: { data: { email: adminUser.email, capabilities: ['commission.read'] } },
    }))
    await page.route('**/api/admin/finance/summary*', route => route.fulfill({
      json: { data: {
        period_start: '2026-10-06T00:00:00Z', period_end: '2026-10-13T00:00:00Z',
        one_time_paid: [{ currency: 'USD', amount_minor: 1000 }],
        subscription_paid: [], subscription_refunded: [], net_product_revenue: [],
        partner_commission_net: [{ currency: 'USD', amount_minor: 200 }],
        partner_payout_paid: [], partner_pending_payout: [],
      } },
    }))

    await page.goto('/app/admin/finance')
    await expect(page.getByRole('heading', { name: '财务' })).toBeVisible()
    await expect(page.getByText('一次性收入')).toBeVisible()
    await expect(page.getByText('USD 10.00')).toBeVisible()
    await expect(page.getByText('USD 2.00')).toBeVisible()
  })

  test('grants a templated entitlement compensation', async ({ page }) => {
    const compensations = []
    await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
    await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
    await page.route('**/api/admin/session', route => route.fulfill({
      json: { data: { email: adminUser.email, capabilities: ['entitlement.read', 'entitlement.manage'] } },
    }))
    await page.route('**/api/admin/entitlements/compensation', async (route) => {
      compensations.push(route.request().postDataJSON())
      await route.fulfill({
        status: 201,
        json: {
          data: {
            template: 'support_30d', entitlement: { id: 'ent_new' }, email_queued: true,
          },
        },
      })
    })
    await page.route('**/api/admin/entitlements*', route => route.fulfill({ json: { data: { entitlements: [] } } }))

    await page.goto('/app/admin/entitlements')
    await page.locator('.compensation-card').getByPlaceholder('user@example.com').fill('customer@example.com')
    await page.locator('.compensation-card .el-select').click()
    await page.getByRole('option', { name: '补偿 30 天' }).click()
    await page.getByPlaceholder('服务事故 / 工单编号').fill('ticket-1')
    await page.getByRole('button', { name: '发放补偿' }).click()

    await expect(page.getByText('补偿已发放')).toBeVisible()
    expect(compensations).toEqual([{
      email: 'customer@example.com', template: 'support_30d', reason: 'ticket-1',
      idempotency_key: expect.stringMatching(/^[0-9a-f-]{36}$/i),
    }])
  })
})

test.describe('Vue admin administrators', () => {
  test('mobile card renders the status tag instead of the raw code', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
    await page.route('**/api/me', route => route.fulfill({ json: { data: adminUser } }))
    await page.route('**/api/admin/session', route => route.fulfill({
      json: { data: { email: adminUser.email, capabilities: ['admin.user.manage'] } },
    }))
    await page.route('**/api/admin/admins', route => route.fulfill({
      json: { data: { admins: [{
        id: 'admin_1', email: 'ops@example.com', status: 'disabled',
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
        capabilities: [{ capability: 'sales.read', granted_at: '2026-10-01T00:00:00Z' }],
      }] } },
    }))

    await page.goto('/app/admin/administrators')

    const card = page.locator('.admin-record-card')
    await expect(card).toContainText('ops@example.com')
    await expect(card.locator('.el-tag').filter({ hasText: '已禁用' })).toBeVisible()
    await expect(card).not.toContainText('disabled')
  })
})
