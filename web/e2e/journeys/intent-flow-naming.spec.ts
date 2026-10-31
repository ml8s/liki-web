import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/intent-flow'
import { mockEngineTool } from '../mocks'
import { baziChart } from '../engine-fixtures'

const yongShen = {
  selected: { yong: '木', xi: ['水', '木'], ji: ['土'], source: 'tiao_hou', confidence: 'high' },
  status: 'ok',
  rule_id: 'YS-TH-1',
}

test('computes the naming direction artifact from the selected profile', async ({ page }) => {
  await mockFlowWorkspace(page, { profiles: [flowProfile('person-1', '张三')] })
  await mockEngineTool(page, 'bazi_chart', baziChart)
  await mockEngineTool(page, 'bazi_yongshen', yongShen)

  await page.goto('/app/?group=naming&intent=naming')
  const form = page.locator('.flow-naming-form')
  await expect(form).toBeVisible()
  await expect(form).toContainText('张三')
  await form.getByRole('button', { name: '起名' }).click()

  await expect(page.locator('.artifact-capsule')).toBeVisible()
  await page.locator('.artifact-capsule').click()
  await expect(page.locator('.engine-yongshen')).toBeVisible()
  await expect(page.locator('.ys-value')).toHaveText('木')
})
