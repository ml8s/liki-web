import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'
import { fillBirthFields } from '../helpers'
import { mockEngineTool } from '../mocks'
import { baziChart, bond } from '../engine-fixtures'

test('computes the natal hepan focused form and renders its artifact', async ({ page }) => {
  await mockFlowWorkspace(page, { profiles: [flowProfile('person-1', '张三')] })
  await mockEngineTool(page, 'bazi_chart', baziChart)
  await mockEngineTool(page, 'bazi_bond', bond)

  await page.goto('/app/?domain=natal&seed=hepan')
  await expect(page.locator('.flow-hepan-form')).toBeVisible()

  const form = page.locator('.flow-hepan-form')
  await fillBirthFields(page, form, {
    date: '1992-08-20',
    time: '10:15',
    gender: 'male',
  })
  await form.locator('.region-manual-toggle').click()
  await form.locator('input[name="flow-hepan-other-longitude"]').fill('121.4726')
  await form.locator('input[name="flow-hepan-other-timezone"]').fill('Asia/Shanghai')
  await expect(form.locator('.el-form-item__error')).toHaveCount(0)

  await form.locator('.el-checkbox').click()
  await form.getByRole('button', { name: '合盘' }).click()

  const capsule = page.locator('.artifact-capsule')
  await expect(capsule).toBeVisible()
  await capsule.click()

  const panel = page.locator('.engine-bond')
  await expect(panel).toBeVisible()
  await expect(panel.locator('.pillars')).toHaveCount(2)

  await page.getByRole('button', { name: '保存记录' }).click()
  await expect(page.locator('.artifact-capsule')).toBeVisible()
})
