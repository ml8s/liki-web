// @ts-check
// 编辑保存的反馈：版本冲突（其他设备已改）必须看到原因且保留草稿；保存成功才关闭表单。
// 加载失败有独立页面态，这里覆盖的是「编辑态内」的失败路径。
import { test, expect } from '@playwright/test'

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const user = { id: 'usr_profile', email: 'profile@example.com', status: 'active', locale: 'zh-Hant' }
const profile = {
  id: 'p_1', user_id: user.id, version: 3, nickname: '媽媽', name: '', gender: 'female',
  calendar: 'solar', birth_date: '1958-05-12', birth_time: '06:00', birth_time_known: 'known',
  birth_place: '上海', note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

async function mockDetail(page, onPatch) {
  await page.route('**/api/**', route => route.fulfill(json({})))
  await page.route('**/api/me', route => route.fulfill(json(user)))
  await page.route('**/api/me/entitlements', route => route.fulfill(json({ entitlements: [], allowed_skills: [] })))
  await page.route('**/api/birth-profiles/p_1', route => (
    route.request().method() === 'PATCH' ? onPatch(route) : route.fulfill(json(profile))
  ))
}

async function startEditing(page) {
  await page.goto('/app/birth-profiles/p_1')
  await expect(page.getByRole('heading', { name: '媽媽' })).toBeVisible()
  await page.getByRole('button', { name: '編輯' }).click()
  await page.getByLabel(/稱呼/).fill('媽媽（新）')
}

test('a conflicting save keeps the form and explains the conflict', async ({ page }) => {
  const patches = []
  await mockDetail(page, route => {
    patches.push(route.request().postDataJSON())
    return route.fulfill({
      status: 409,
      contentType: 'application/problem+json',
      body: JSON.stringify({ type: 'profile_conflict', title: 'profile_conflict', status: 409, detail: 'conflict' }),
    })
  })

  await startEditing(page)
  await page.getByRole('button', { name: '儲存檔案' }).click()

  await expect(page.locator('.profile-edit-error')).toContainText('檔案已被其他裝置修改')
  // 仍在编辑态：草稿与冲突提示同时可见，用户刷新后可重试。
  await expect(page.getByRole('button', { name: '儲存檔案' })).toBeVisible()
  await expect(page.getByLabel(/稱呼/)).toHaveValue('媽媽（新）')
  expect(patches).toEqual([expect.objectContaining({ expected_version: 3, nickname: '媽媽（新）' })])
})

test('a successful save closes the form and shows the new value', async ({ page }) => {
  await mockDetail(page, route => route.fulfill(json({ ...profile, version: 4, nickname: '媽媽（新）' })))

  await startEditing(page)
  await page.getByRole('button', { name: '儲存檔案' }).click()

  await expect(page.getByText('v4')).toBeVisible()
  await expect(page.locator('.profile-edit-error')).toBeHidden()
  await expect(page.getByRole('button', { name: '編輯' })).toBeVisible()
})
