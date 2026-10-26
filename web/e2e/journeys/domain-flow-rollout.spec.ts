import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'

const profile = flowProfile('person-1', '张三')

const redirects = [
  ['/app/hepan', '.flow-hepan-form', { domain: 'natal', seed: 'hepan' }],
  ['/app/divination', '.flow-domain-strip', { domain: 'divination', seed: 'divination' }],
  ['/app/fengshui', '.flow-fengshui-form', { domain: 'fengshui', seed: 'fengshui' }],
  ['/app/liunian', '.flow-domain-strip', { domain: 'natal', seed: 'liunian' }],
  ['/app/naming', '.flow-naming-form', { domain: 'naming', seed: 'naming' }],
] as const

test.describe('phase six rollout redirects', () => {
  for (const [from, marker, query] of redirects) {
    test(`redirects ${from} into Consultation`, async ({ page }) => {
      await mockFlowWorkspace(page, { profiles: [profile] })
      await page.goto(`${from}?locale=zh-Hans`)

      await expect(page).toHaveURL(new RegExp(`domain=${query.domain}&seed=${query.seed}`))
      await expect(page.locator(marker)).toBeVisible()
    })
  }

  test('keeps pending-backend huangli on its compatibility route', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles: [profile] })
    await page.goto('/app/huangli?locale=zh-Hans')

    await expect(page).toHaveURL(/\/app\/classic\/huangli(?:\?|$)/)
    await expect(page.locator('.el-select').first()).toBeVisible()
  })
})
