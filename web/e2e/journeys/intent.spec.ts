import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { json, mockPageview, mockWorkspace, testUser } from '../mocks'


// 现行场景落地页：naming 独立页 + natal 枢纽 + hepan 独立页。
const pages = [
  { path: '/zh-Hans/naming/', heading: 'AI 起名与姓名解读', cta: 'naming', href: '/app?intent=naming&locale=zh-Hans' },
  { path: '/zh-Hant/natal/', heading: '線上命理：八字、紫微與命書', cta: 'natal', href: '/app?service=natal&locale=zh-Hant' },
  { path: '/en/natal/', heading: 'Chinese Astrology: BaZi, Zi Wei Dou Shu & Life Reading', cta: 'natal', href: '/app?service=natal&locale=en' },
  { path: '/zh-Hans/hepan/', heading: '八字合盘：合婚、姻缘与关系解读', cta: 'hepan', href: '/app?intent=hepan&locale=zh-Hans' },
  { path: '/en/hepan/', heading: 'BaZi Compatibility: Marriage Matching & Relationship Reading', cta: 'hepan', href: '/app?intent=hepan&locale=en' },
  { path: '/zh-Hans/liunian/', heading: '流年运势：这一年，你的机会与注意', cta: 'liunian', href: '/app?intent=liunian&locale=zh-Hans' },
]

test.describe('Intent landing pages', () => {
  test.beforeEach(async ({ page }) => {
    await mockPageview(page)
  })

  for (const { path, heading, cta, href } of pages) {
    test(`${path} routes to Liki Chat`, async ({ page }) => {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible()
      await expect(page.locator(`[data-cta="${cta}"]`).first()).toHaveAttribute('href', href)
      // 落地页不得出现假结账表单（工具岛的真实表单由各自工具 spec 覆盖）。
      await expect(page.locator('[data-price], input[type="email"]')).toHaveCount(0)
    })
  }

})

test.describe('Authenticated intent handoff', () => {
  const user = testUser()
  const profile = {
    id: 'p_1', user_id: 'usr_test', version: 1, nickname: 'Mom', name: '',
    birth: {
      gender: 'female', date: '1990-01-01', time: '06:00',
      instant: '1989-12-31T22:00:00Z',
      location: { place: 'Shanghai', longitude: 121.4737, timezone: 'Asia/Shanghai' },
    },
    note: '', third_party_consent: true, last_used_at: null,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }

  async function mockAuthenticated(page: Page, { profiles = [profile] }: { profiles?: unknown[] } = {}) {
    await mockWorkspace(page, { user, allowedSkills: ['liki'], profiles })
    await page.route('**/api/me/sessions', route => route.fulfill(json([])))
    await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))
  }

  test('naming and hepan reach their App workspaces in English', async ({ page }) => {
    await mockAuthenticated(page)
    await page.goto('/app/hepan?locale=en')
    await expect(page).toHaveURL('/app/hepan?locale=en')
    await expect(page.getByRole('heading', { level: 2, name: 'Step 1 · Select a profile (you or the other person)' }).first())
      .toBeVisible()
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')
  })

})

test('unauthenticated intent handoff preserves locale and redirect', async ({ page }) => {
  await page.route('**/api/me', route => route.fulfill({
    status: 401,
    contentType: 'application/json',
    body: JSON.stringify({ error: { code: 'unauthorized', message: 'unauthorized' } }),
  }))
  await page.goto('/app?intent=naming&locale=en')
  await expect(page).toHaveURL(/\/app\/login/)
  await expect(page.getByRole('heading', { level: 1, name: 'Sign in to Liki' })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('lang', 'en')
  const redirect = new URL(page.url()).searchParams.get('redirect') ?? ''
  expect(redirect).toBe('/?intent=naming&locale=en')
  expect(new URL(page.url()).searchParams.get('locale')).toBe('en')
})
