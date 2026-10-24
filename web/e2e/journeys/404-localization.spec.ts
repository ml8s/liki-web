import { test, expect } from '../fixtures'

// The static 404 page is served by the Go file server for every locale. Its
// small external script selects the locale from the path; HTTP status must
// stay 404 while the copy follows /en/ and /zh-Hans/.
const CASES = [
  { path: '/en/smoke-not-found', lang: 'en', heading: 'Page not found' },
  { path: '/zh-Hans/smoke-not-found', lang: 'zh-Hans', heading: '页面未找到' },
  { path: '/smoke-not-found', lang: 'zh-Hant', heading: '頁面未找到' },
] as const

test.describe('404 localization', () => {
  for (const item of CASES) {
    test(`${item.path} keeps 404 and localizes copy`, async ({ page }) => {
      const response = await page.goto(item.path)
      expect(response?.status()).toBe(404)
      await expect(page.locator('html')).toHaveAttribute('lang', item.lang)
      await expect(page.locator('[data-404-heading]')).toHaveText(item.heading)
      await expect(page.locator('[data-404-cta]')).toBeVisible()
    })
  }
})
