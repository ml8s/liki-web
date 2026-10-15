// 登录全程旅程：未登录带场景访问 → 邮箱验证码登录 → 应回到原始场景。
// 防回归：redirect 曾经丢 base 前缀，登录后落到 /app/app（404 视图）。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { json, testUser } from '../mocks'


const user = testUser({ id: 'usr_login_journey', email: 'login-journey@example.com' })

async function mockLoginApis(page: Page) {
  await page.route('**/api/me', route => route.fulfill({
    status: 401,
    contentType: 'application/json',
    body: JSON.stringify({ type: 'about:blank', title: 'unauthenticated', status: 401 }),
  }))
  await page.route('**/api/auth/email/code', route => route.fulfill(json({})))
  await page.route('**/api/auth/email/verify', route => route.fulfill(json({
    user,
    session_id: 'sess_login_journey',
    expires_at: '2027-01-01T00:00:00Z',
  })))
  // Post-login workspace loads; content is irrelevant to the redirect assertion.
  await page.route('**/api/me/entitlements', route => route.fulfill(json({ entitlements: [], allowed_skills: [] })))
  await page.route('**/api/birth-profiles**', route => route.fulfill(json({ profiles: [], next_cursor: '' })))
  await page.route('**/api/chat/sessions', route => route.fulfill(json([])))
  await page.route('**/api/sales/products', route => route.fulfill(json({ products: [] })))
  await page.route('**/api/sales/orders', route => route.fulfill(json({ orders: [] })))
  await page.route('**/api/analytics/pageview', route => route.fulfill(json({})))
}

async function login(page: Page) {
  await page.getByLabel(/郵箱|邮箱|email/i).fill(`journey-${Date.now()}@example.com`)
  await page.getByRole('button', { name: /取得|获取|get code/i }).click()
  await page.getByLabel(/驗證碼|验证码|verification code/i).fill('123456')
}

test.describe('Login journey', () => {
  test('scenario handoff survives login', async ({ page }) => {
    await mockLoginApis(page)

    // Marketing service card target: unauthenticated user lands on login.
    await page.goto('/app?scenario=naming&locale=zh-Hant')
    await expect(page).toHaveURL(/\/app\/login/)

    await login(page)

    // The dashboard consumes ?scenario and may advance to /chat or profile
    // creation; the invariant is: back inside the app, never /app/app.
    await expect(page).toHaveURL(url => {
      const u = new URL(url)
      return u.pathname.startsWith('/app')
        && !u.pathname.startsWith('/app/app')
        && !u.pathname.startsWith('/app/login')
    })
  })

  test('plain login lands on the dashboard', async ({ page }) => {
    await mockLoginApis(page)

    await page.goto('/app/login')
    await login(page)

    await expect(page).toHaveURL(url => {
      const u = new URL(url)
      return u.pathname === '/app' || u.pathname === '/app/'
    })
  })
})
