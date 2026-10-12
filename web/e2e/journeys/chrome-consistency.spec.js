// @ts-check
// 骨架一致性：全站每个静态页共享同一 header（brand link + slogan，无返回链）
// 与同一 footer（探索/法律两组）。差异化只允许发生在正文区。
import { test, expect } from '@playwright/test'

const LOCALES = ['zh-Hant', 'zh-Hans', 'en']
const PAGES = ['', 'skills', 'naming', 'mingshu', 'hepan', 'partners', 'about', 'contact', 'disclaimer', 'privacy', 'terms']

test.describe.configure({ mode: 'serial' })

for (const locale of LOCALES) {
  for (const p of PAGES) {
    const path = `/${locale}/${p ? p + '/' : ''}`
    test(`${path} chrome`, async ({ page }) => {
      await page.goto(path)
      // header：brand logo（链接首页）+ slogan，且没有返回链残留。
      await expect(page.locator('header .brand-link')).toHaveText('Liki')
      await expect(page.locator('.brand-slogan')).toBeVisible()
      await expect(page.locator('.back-link')).toHaveCount(0)
      // footer：探索/法律两组标题可见，7 个链接齐备。
      await expect(page.locator('.footer-group-label')).toHaveText([/探索|Explore/i, /法律|Legal/i])
      await expect(page.locator('footer a[href$="/partners/"], footer a[href$="/partners"]')).toHaveCount(1)
      await expect(page.locator('footer a[href$="/privacy/"], footer a[href$="/privacy"]')).toHaveCount(1)
    })
  }
}
