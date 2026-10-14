// @ts-check
import { test, expect } from '@playwright/test'

const json = data => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ data }),
})

const pages = [
  { locale: 'zh-Hans', scenario: 'naming', path: '/zh-Hans/naming/', heading: 'AI 起名与姓名解读' },
  { locale: 'zh-Hant', scenario: 'mingshu', path: '/zh-Hant/mingshu/', heading: '個人命書與終身運勢解讀' },
  { locale: 'en', scenario: 'hepan', path: '/en/hepan/', heading: 'BaZi compatibility and relationship reading' },
]

test.describe('Scenario landing pages', () => {
  for (const { locale, scenario, path, heading } of pages) {
    test(`${locale}/${scenario} routes to Liki Chat`, async ({ page }) => {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible()
      await expect(page.locator(`[data-cta="${scenario}"]`).first()).toHaveAttribute(
        'href',
        `/app?scenario=${scenario}&locale=${locale}`,
      )
      await expect(page.locator('form, input[type="email"], [data-price]')).toHaveCount(0)
    })
  }
})

test.describe('Authenticated scenario handoff', () => {
  const user = {
    id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant',
  }
  const profile = {
    id: 'p_1', user_id: 'usr_test', version: 1, nickname: 'Mom', name: '',
    gender: 'female', calendar: 'solar', birth_date: '1990-01-01',
    birth_time: '06:00', birth_time_known: 'known', birth_place: 'Shanghai',
    note: '', third_party_consent: true, last_used_at: null,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }

  async function mockAuthenticated(page, { profiles = [profile] } = {}) {
    await page.route('**/api/me', route => route.fulfill(json(user)))
    await page.route('**/api/me/entitlements', route => route.fulfill(json({
      entitlements: [], allowed_skills: ['liki'],
    })))
    await page.route('**/api/birth-profiles**', route => route.fulfill(json({ profiles, next_cursor: '' })))
    await page.route('**/api/chat/sessions', route => route.fulfill(json([])))
    await page.route('**/api/sales/products', route => route.fulfill(json({ products: [] })))
    await page.route('**/api/sales/orders', route => route.fulfill(json({ orders: [] })))
    await page.route('**/api/me/sessions', route => route.fulfill(json([])))
    await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))
  }

  test('naming and hepan reach their App workspaces in English', async ({ page }) => {
    await mockAuthenticated(page)
    await page.goto('/app?scenario=naming&locale=en')
    await expect(page).toHaveURL(/\/app\/chat/)
    await expect(page.getByRole('heading', { level: 1, name: 'Liki Chat' })).toBeVisible()
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')

    await page.goto('/app?scenario=hepan&locale=en')
    await expect(page).toHaveURL(/\/app\/synastry/)
    await expect(page.getByRole('heading', { level: 1, name: 'Compatibility' })).toBeVisible()
  })

  test('mingshu without a profile guides profile creation', async ({ page }) => {
    await mockAuthenticated(page, { profiles: [] })
    await page.goto('/app?scenario=mingshu&locale=en')
    await expect(page).toHaveURL(/\/app\/profiles\/new/)
    await expect(page.getByRole('heading', { level: 1, name: 'New profile' })).toBeVisible()
    expect(await page.evaluate(() => sessionStorage.getItem('liki-service'))).toBe('bazi')
  })
})

test('unauthenticated scenario handoff preserves locale and redirect', async ({ page }) => {
  await page.route('**/api/me', route => route.fulfill({
    status: 401,
    contentType: 'application/json',
    body: JSON.stringify({ error: { code: 'unauthorized', message: 'unauthorized' } }),
  }))
  await page.goto('/app?scenario=naming&locale=en')
  await expect(page).toHaveURL(/\/app\/login/)
  await expect(page.getByRole('heading', { level: 1, name: 'Sign in to Liki' })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('lang', 'en')
  const redirect = new URL(page.url()).searchParams.get('redirect') ?? ''
  expect(redirect).toBe('/?scenario=naming&locale=en')
  expect(new URL(page.url()).searchParams.get('locale')).toBe('en')
})
