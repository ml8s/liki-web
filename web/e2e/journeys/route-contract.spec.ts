import { expect, test } from '../fixtures'
import { mockPageview } from '../mocks'
import marketingContract from '../../routes/marketing.contract.json' with { type: 'json' }
import intentContract from '../../shared/config/intents.json' with { type: 'json' }

const slugs = intentContract.intents.map(intent => intent.marketingSlug)

test.beforeEach(async ({ page }) => {
  await mockPageview(page)
})

test.describe('marketing route contract', () => {
  for (const locale of marketingContract.locales) {
    for (const slug of slugs) {
      test(`${locale}/${slug}/ is canonical`, async ({ page }) => {
        const response = await page.goto(`/${locale}/${slug}/`)
        expect(response?.status()).toBe(200)
        await expect(page.locator('h1')).toBeVisible()
      })
    }
  }

  for (const route of marketingContract.retired) {
    test(`${route.path} remains retired`, async ({ request }) => {
      const response = await request.get(route.path, { maxRedirects: 0 })
      expect(response.status(), `${route.path} location=${response.headers().location ?? ''}`).toBe(route.status)
    })
  }
})
