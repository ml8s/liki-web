import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/intent-flow'

test('opens the Me hub and navigates to existing profile management', async ({ page }) => {
  await mockFlowWorkspace(page, { profiles: [flowProfile('person-1', '张三')] })

  await page.goto('/app/me')
  await expect(page.getByRole('heading', { name: '我的' })).toBeVisible()

  const entries = page.locator('.entry-row')
  await expect(entries).toHaveCount(4)
  await expect(page.getByText('我的合作')).toHaveCount(0)

  await entries.first().click()
  await expect(page).toHaveURL(/\/app\/classic\/birth-profiles$/)
  await expect(page.getByRole('heading', { name: '档案', level: 1 })).toBeVisible()
})
