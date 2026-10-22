// 合作入口隔离守卫（定向邀请制）：
// - KOL 侧全链路（邀请页 / 申请面 / 中心页）零 "Scout" 泄漏；
// - 普通用户在 App 任何位置看不到合作入口；
// - 合作方（partner_roles 非空）看得到"我的合作"，且中心页只显示自己的角色。
import { test, expect } from '../fixtures'
import type { Page } from '@playwright/test'
import { json, mockPageview, mockWorkspace, testUser } from '../mocks'

const LOCALES = ['zh-Hans', 'zh-Hant', 'en']

const statesFor = (role: 'kol' | 'scout', status: string) => ({
  kol: { role: 'kol', status: role === 'kol' ? status : '', application: null, can_submit: role !== 'kol' },
  scout: { role: 'scout', status: role === 'scout' ? status : '', application: null, can_submit: role !== 'scout' },
})

async function mockPartnerWorkspace(page: Page, partnerRoles: string[]) {
  await mockWorkspace(page, { user: testUser({ partner_roles: partnerRoles }) })
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
}

test.describe('KOL invite pages never mention Scout', () => {
  for (const locale of LOCALES) {
    test(`${locale}/partner/kol`, async ({ page }) => {
      await mockPageview(page)
      await page.goto(`/${locale}/partner/kol/`)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const html = await page.content()
      expect(html.toLowerCase(), 'KOL 邀请页不得出现 Scout 字样').not.toContain('scout')
    })
  }
})

test('Scout invite page exists and routes to the scout role page', async ({ page }) => {
  await mockPageview(page)
  await page.goto('/zh-Hant/partner/scout/')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await expect(page.locator('a.btn-purple').first()).toHaveAttribute('href', '/app/partners/scout?locale=zh-Hant')
})

test('regular user sees no partnership entry anywhere', async ({ page }) => {
  await mockPartnerWorkspace(page, [])

  await page.goto('/app/')
  await expect(page.getByRole('link', { name: '我的合作' })).toHaveCount(0)

  await page.goto('/app/account')
  // el-button tag="router-link" 渲染为 <a>：与侧栏链接同角色，普通用户两者都不应存在。
  await expect(page.getByRole('link', { name: '我的合作' })).toHaveCount(0)
})

test('partner user sees the entry; the center shows only their own role', async ({ page }) => {
  await mockPartnerWorkspace(page, ['kol'])
  await page.route('**/api/partner-portal', r => r.fulfill(json(statesFor('kol', 'active'))))

  await page.goto('/app/')
  await expect(page.getByRole('link', { name: '我的合作' }).first()).toBeVisible()

  await page.goto('/app/partners')
  await expect(page.locator('.role-card')).toHaveCount(1)
  await expect(page.locator('.role-card')).toContainText('KOL')
  const text = await page.locator('main').innerText()
  expect(text.toLowerCase(), '中心页不得出现 Scout 字样').not.toContain('scout')
})

test('direct visit without a role shows the neutral invitation-only state', async ({ page }) => {
  await mockPartnerWorkspace(page, [])
  await page.route('**/api/partner-portal', r => r.fulfill(json({
    kol: { role: 'kol', status: '', application: null, can_submit: true },
    scout: { role: 'scout', status: '', application: null, can_submit: true },
  })))

  await page.goto('/app/partners')
  await expect(page.locator('.el-empty')).toBeVisible()
  await expect(page.locator('.role-card')).toHaveCount(0)
})

test('KOL role page never mentions Scout', async ({ page }) => {
  await mockPartnerWorkspace(page, ['kol'])
  await page.route('**/api/partner-portal', r => r.fulfill(json(statesFor('kol', 'active'))))
  await page.route('**/api/partner-portal/kol/**', r => r.fulfill(json({})))

  await page.goto('/app/partners/kol')
  await expect(page.locator('main')).toBeVisible()
  const text = await page.locator('main').innerText()
  expect(text.toLowerCase(), 'KOL 角色页不得出现 Scout 字样').not.toContain('scout')
})
