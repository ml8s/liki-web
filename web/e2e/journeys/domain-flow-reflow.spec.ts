import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'
import { expectReflowContract } from '../helpers/reflow'

const profile = flowProfile('person-1', '张三')

test.describe('consultation reflow', () => {
  for (const width of [320, 640, 768, 1024, 1440]) {
    test(`keeps primary controls usable at ${width}px (200% reference)`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 })
      await mockFlowWorkspace(page, { profiles: [profile] })
      await page.goto('/app/')
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()

      await expectReflowContract(page, {
        viewportWidth: width,
        scopeSelector: '.flow-domain-strip',
        controlSelectors: [
          '.flow-domain-strip',
          '.flow-parameters button:has-text("张三")',
          '.flow-parameters button:has-text("命理")',
          'textarea[name="flow-question"]',
          'button.flow-send',
          '.experience-switcher--dock:has-text("我的")',
        ],
      })
    })
  }

  test('keeps the Me hub usable at 320px', async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 900 })
    await mockFlowWorkspace(page, { profiles: [profile] })
    await page.goto('/app/me')
    await expect(page.locator('.entry-row')).toHaveCount(4)

    await expectReflowContract(page, {
      viewportWidth: 320,
      scopeSelector: '.page-stack',
      controlSelectors: ['.entry-row'],
    })
  })
})
