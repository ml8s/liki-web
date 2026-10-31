import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/intent-flow'

const profile = flowProfile('person-1', '张三')

const redirects = [
  ['/app/hepan', '.flow-hepan-form', { intentGroup: 'natal', seed: 'hepan' }],
  ['/app/divination', '.flow-divination-form', { intentGroup: 'divination', seed: 'divination' }],
  ['/app/fengshui', '.flow-fengshui-form', { intentGroup: 'fengshui', seed: 'fengshui' }],
  ['/app/liunian', '.flow-main', { intentGroup: 'natal', seed: 'liunian' }],
  ['/app/naming', '.flow-naming-form', { intentGroup: 'naming', seed: 'naming' }],
] as const

test.describe('phase six rollout redirects', () => {
  for (const [from, marker, query] of redirects) {
    test(`redirects ${from} into Consultation`, async ({ page }) => {
      const productEvents: Array<Record<string, unknown>> = []
      await mockFlowWorkspace(page, {
        profiles: [profile],
        productEvents,
      })
      await page.goto(`${from}?locale=zh-Hans`)

      await expect(page).toHaveURL(new RegExp(`group=${query.intentGroup}&intent=${query.seed}`))
      await expect(page.locator(marker)).toBeVisible()
      await expect.poll(() => productEvents).toContainEqual({
        event: 'consultation_entry',
        intentGroup: query.intentGroup,
        source: 'legacy_redirect',
        intent: query.seed,
        status: 'empty',
      })
    })
  }

  test('keeps pending-backend huangli on its compatibility route', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles: [profile] })
    await page.goto('/app/huangli?locale=zh-Hans')

    await expect(page).toHaveURL(/\/app\/classic\/huangli(?:\?|$)/)
    await expect(page.locator('.el-select').first()).toBeVisible()
  })
})
