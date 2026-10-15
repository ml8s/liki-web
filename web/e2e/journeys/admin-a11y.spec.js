// @ts-check
import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const adminUser = { id: 'usr_admin', email: 'admin@example.com', status: 'active', locale: 'zh-Hant' }
const capabilities = [
  'entitlement.read', 'entitlement.manage', 'partner.read', 'partner.approve', 'partner.freeze',
  'partner.unfreeze', 'commission.read', 'commission.void', 'payout.read', 'payout.approve',
  'payout.reject', 'payout.mark_paid', 'audit.read', 'admin.user.manage', 'admin.super',
]
const meta = { page: 1, page_size: 50, total: 0, has_more: false }
const financeReport = {
  period_start: '2026-10-01T00:00:00Z', period_end: '2026-10-31T00:00:00Z',
  one_time_paid: [], subscription_paid: [], subscription_refunded: [], net_product_revenue: [],
  partner_commission_net: [], partner_payout_paid: [], partner_pending_payout: [],
}

const pages = [
  { path: '/app/admin', name: 'admin-overview', marker: '.admin-nav' },
  { path: '/app/admin/entitlements', name: 'admin-entitlements', marker: '.filters' },
  { path: '/app/admin/partners', name: 'admin-partners', marker: '.admin-page-heading' },
  { path: '/app/admin/payouts', name: 'admin-payouts', marker: '.admin-page-heading' },
  { path: '/app/admin/finance', name: 'admin-finance', marker: '.admin-shell' },
  { path: '/app/admin/audit', name: 'admin-audit', marker: '.admin-shell' },
  { path: '/app/admin/administrators', name: 'admin-administrators', marker: '.toolbar-card' },
]

test.describe('Admin accessibility audit', () => {
  for (const { path, name, marker } of pages) {
    test(`${name}: ${path}`, async ({ page }) => {
      await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }))
      await page.route('**/api/me', route => route.fulfill(json(adminUser)))
      await page.route('**/api/admin/session', route => route.fulfill(json({ email: adminUser.email, capabilities })))
      await page.route('**/api/admin/entitlements*', route => route.fulfill(json({ entitlements: [] })))
      await page.route('**/api/admin/partner-applications*', route => route.fulfill(json({ applications: [], ...meta })))
      await page.route('**/api/admin/partners*', route => route.fulfill(json({ partners: [], ...meta })))
      await page.route('**/api/admin/partner-payouts*', route => route.fulfill(json({ payouts: [], ...meta })))
      await page.route('**/api/admin/partner-audit-actions*', route => route.fulfill(json({ actions: [], ...meta })))
      await page.route('**/api/admin/partner-finance-tasks*', route => route.fulfill(json({ tasks: [], ...meta })))
      await page.route('**/api/admin/finance/summary*', route => route.fulfill(json(financeReport)))
      await page.route('**/api/admin/admins*', route => route.fulfill(json({ admins: [] })))

      await page.goto(path)
      await page.waitForSelector(marker, { timeout: 10000 })
      await page.waitForTimeout(250)

      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .analyze()
      const criticalSerious = results.violations.filter(v => v.impact === 'critical' || v.impact === 'serious')
      expect(criticalSerious, `a11y violations on ${path}:\n` + criticalSerious
        .map(v => `${v.id}: ${v.nodes.length} nodes [${v.nodes.map(n => n.target.join(' ')).join('; ')}]`)
        .join('\n')).toEqual([])
    })
  }
})
