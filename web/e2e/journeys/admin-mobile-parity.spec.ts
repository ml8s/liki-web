// 后台的移动端不是另一套页面：桌面表格与移动卡片必须给出同样的关键字段。
// 这个套件按视图表驱动，用同一份模拟数据分别断言两种形态，避免「卡片漏字段 /
// 表格显示原始码」这类只在一侧出现的偏差悄悄回归。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { mockAdmin } from '../mocks'

const pageMeta = { page: 1, page_size: 50, total: 1, has_more: false }

async function mockData(page: Page, pattern: string, data: unknown) {
  await page.route(pattern, route => route.fulfill({ json: { data } }))
}

// anchor 必须同时出现在表格行与卡片里；texts 只用标识符、金额、状态标签，
// 不使用日期/时间（随时区与语言变化）。table 默认取页面上所有表格，页面另有
// 非记录类表格（如财务周期汇总）时按用例收窄。
interface AdminCase {
  name: string
  path: string
  capabilities: string[]
  api: [string, unknown][]
  anchor: string
  texts: string[]
  table?: string
  prepare?: (page: Page) => Promise<void>
}

const cases: AdminCase[] = [
  {
    name: '订单',
    path: '/app/admin/orders',
    capabilities: ['sales.read'],
    api: [['**/api/admin/sales/orders*', {
      ...pageMeta,
      orders: [{
        id: 'order_1', user_id: 'usr_1', email: 'customer@example.com', product_id: 'liki-chat',
        amount_minor: 12300, currency: 'USD', status: 'paid', payment_intent_id: 'pi_1',
        created_at: '2026-10-01T00:00:00Z', paid_at: '2026-10-01T00:05:00Z',
      }],
    }]],
    anchor: 'order_1',
    texts: ['customer@example.com', 'USD 123.00', '已支付'],
  },
  {
    name: '订阅',
    path: '/app/admin/subscriptions',
    capabilities: ['sales.read'],
    api: [['**/api/admin/sales/subscriptions*', {
      ...pageMeta,
      subscriptions: [{
        id: 'sub_1', user_id: 'usr_1', email: 'customer@example.com', product_id: 'liki-chat',
        billing_mode: 'subscription', provider: 'dodo', provider_subscription_id: 'dodo_sub_1',
        status: 'on_hold', currency_code: 'USD', cancel_at_period_end: false,
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
      }],
    }]],
    anchor: 'sub_1',
    texts: ['dodo_sub_1', '暂停'],
  },
  {
    name: '提现',
    path: '/app/admin/payouts',
    capabilities: ['payout.read'],
    api: [['**/api/admin/partner-payouts*', {
      ...pageMeta,
      payouts: [{
        id: 'payout_1', beneficiary_id: 'usr_partner', beneficiary_code: 'PAY-CODE', role: 'kol',
        amount_minor: 5000, requested_amount_minor: 5000, currency: 'USD', method: 'alipay',
        account: 'partner@example.com', status: 'requested',
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
      }],
    }]],
    anchor: 'PAY-CODE',
    texts: ['USD 50.00', '待审核', 'Alipay'],
  },
  {
    name: '合作申请',
    path: '/app/admin/partners',
    capabilities: ['partner.read', 'partner.approve'],
    api: [
      ['**/api/admin/partner-applications*', {
        ...pageMeta,
        applications: [{
          id: 'app_1', user_id: 'usr_applicant', type: 'kol', code: 'APP-CODE', status: 'approved',
          note: '小红书粉丝 5 万', created_at: '2026-10-01T00:00:00Z',
        }],
      }],
      ['**/api/admin/partners*', { ...pageMeta, total: 0, partners: [] }],
    ],
    anchor: 'APP-CODE',
    texts: ['KOL', '已通过'],
  },
  {
    name: '合作伙伴',
    path: '/app/admin/partners',
    capabilities: ['partner.read', 'partner.freeze', 'partner.unfreeze'],
    api: [
      ['**/api/admin/partner-applications*', { ...pageMeta, total: 0, applications: [] }],
      ['**/api/admin/partners*', {
        ...pageMeta,
        partners: [{
          user_id: 'usr_partner', code: 'PARTNER-CODE', role: 'kol', status: 'active',
          created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
        }],
      }],
    ],
    anchor: 'PARTNER-CODE',
    texts: ['KOL', '有效'],
  },
  {
    name: '权益',
    path: '/app/admin/entitlements',
    capabilities: ['entitlement.read'],
    api: [['**/api/admin/entitlements*', {
      ...pageMeta,
      entitlements: [{
        id: 'ent_1', user_id: 'usr_1', email: 'customer@example.com', product_id: 'liki-chat',
        skills: ['liki'], status: 'active', source: 'payment', source_id: 'order_1',
        starts_at: '2026-10-01T00:00:00Z', expires_at: '2026-11-01T00:00:00Z',
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
      }],
    }]],
    anchor: 'customer@example.com',
    texts: ['liki', '有效'],
    // 权益列表不自动加载，等运营填邮箱后查询。
    prepare: async (page) => {
      await page.getByPlaceholder('user@example.com').fill('customer@example.com')
      await page.getByRole('button', { name: '查询' }).click()
    },
  },
  {
    name: '财务任务',
    path: '/app/admin/finance',
    capabilities: ['commission.read'],
    api: [
      ['**/api/admin/partner-finance-tasks*', {
        ...pageMeta,
        tasks: [{
          id: 'task_1', type: 'commission_refund_clawback', provider: 'dodo', payment_intent_id: 'pi_1',
          provider_refund_id: 'refund_1', entry_id: 'entry_1', beneficiary_id: 'usr_beneficiary',
          beneficiary_role: 'kol', amount_minor: 7000, currency: 'USD', status: 'resolved',
          created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
        }],
      }],
      ['**/api/admin/finance/summary*', {
        period_start: '2026-10-06T00:00:00Z', period_end: '2026-10-13T00:00:00Z',
        one_time_paid: [], subscription_paid: [], subscription_refunded: [], net_product_revenue: [],
        partner_commission_net: [], partner_payout_paid: [], partner_pending_payout: [],
      }],
    ],
    anchor: 'usr_beneficiary',
    texts: ['USD 70.00', '已追回', '佣金退款追回'],
    // 财务页的周期汇总表在窄屏也保留，只有任务列表换成卡片。
    table: '.el-card:has-text("财务任务") .el-table',
  },
  {
    name: '管理员',
    path: '/app/admin/administrators',
    capabilities: ['admin.user.manage'],
    api: [['**/api/admin/admins', {
      admins: [{
        id: 'admin_1', email: 'ops@example.com', status: 'disabled',
        created_at: '2026-10-01T00:00:00Z', updated_at: '2026-10-01T00:00:00Z',
        capabilities: [{ capability: 'sales.read', status: 'active', granted_at: '2026-10-01T00:00:00Z' }],
      }],
    }]],
    anchor: 'ops@example.com',
    texts: ['订单查看', '已禁用'],
  },
  {
    name: '合作伙伴审计',
    path: '/app/admin/audit',
    capabilities: ['partner.read'],
    api: [['**/api/admin/partner-audit-actions*', {
      ...pageMeta,
      actions: [{
        id: 'action_1', actor: 'admin@example.com', action: 'partner.freeze',
        target_type: 'partner', target_id: 'usr_target', reason: '风控复核',
        created_at: '2026-10-01T00:00:00Z',
      }],
    }]],
    anchor: 'usr_target',
    texts: ['partner.freeze', '风控复核'],
  },
]

for (const item of cases) {
  test(`mobile cards keep table parity: ${item.name}`, async ({ page }) => {
    await mockAdmin(page, item.capabilities)
    for (const [pattern, data] of item.api) await mockData(page, pattern, data)

    await page.goto(item.path)
    if (item.prepare) await item.prepare(page)

    const table = page.locator(item.table ?? '.el-table')
    await expect(table.first()).toBeVisible()
    const row = page.locator('.el-table__row', { hasText: item.anchor })
    await expect(row).toBeVisible()
    for (const text of item.texts) await expect(row).toContainText(text)
    await expect(page.locator('.admin-record-card')).toHaveCount(0)

    await page.setViewportSize({ width: 390, height: 844 })

    const card = page.locator('.admin-record-card', { hasText: item.anchor })
    await expect(card).toBeVisible()
    for (const text of item.texts) await expect(card).toContainText(text)
    await expect(table).toHaveCount(0)
  })
}
