// 骨架一致性：全站每个静态页共享同一 header（brand link + slogan，无返回链）
// 与同一 footer（两道链接行：站点 / 法律，无分组标题）。差异化只允许发生在正文区。
// 合作计划为定向邀请制：footer 不得出现其入口。
import { test, expect } from '../fixtures'
import { mockPageview } from '../mocks'

const LOCALES = ['zh-Hant', 'zh-Hans', 'en']
const PAGES = ['', 'naming', 'natal', 'partner/kol', 'partner/scout', 'about', 'contact', 'disclaimer', 'privacy', 'terms']

test.describe.configure({ mode: 'serial' })

test.beforeEach(async ({ page }) => {
  await mockPageview(page)
})

for (const locale of LOCALES) {
  for (const p of PAGES) {
    const path = `/${locale}/${p ? p + '/' : ''}`
    test(`${path} chrome`, async ({ page }) => {
      await page.goto(path)
      // header：brand logo（链接首页）+ slogan，且没有返回链残留。
      await expect(page.locator('header .brand-link')).toHaveText('Liki')
      await expect(page.locator('.brand-slogan')).toBeVisible()
      await expect(page.locator('.back-link')).toHaveCount(0)
      // footer：两列链接（无分组标题），7 个链接齐备。
      await expect(page.locator('footer .footer-group')).toHaveCount(2)
      await expect(page.locator('footer .footer-group a')).toHaveCount(6)
      await expect(page.locator('footer a[href$="/partners/"], footer a[href$="/partners"]')).toHaveCount(0)
      await expect(page.locator('footer a[href$="/privacy/"], footer a[href$="/privacy"]')).toHaveCount(1)
      // 标题格式全站统一：`<页面名> · Liki`（静态页与 SPA 同一模式）；
      // 首页的页面名就是 slogan，已经带了站点名，不再追加后缀。
      const title = await page.title()
      if (p === '') {
        // 首页两种合法形态：slogan 本身含品牌名，或「关键词 · Liki」结尾。
        expect(title).toContain('Liki')
      } else {
        expect(title.endsWith(' · Liki'), title).toBe(true)
        expect(title.length, title).toBeGreaterThan(' · Liki'.length)
      }
    })
  }
}
