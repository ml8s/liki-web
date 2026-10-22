// 编辑保存的反馈：版本冲突（其他设备已改）必须看到原因且保留草稿；保存成功才关闭表单。
// 加载失败有独立页面态，这里覆盖的是「编辑态内」的失败路径。
import { test, expect } from '../fixtures'
import type {Page, Route} from '@playwright/test'
import { json, mockWorkspace, testUser } from '../mocks'

const user = testUser({ id: 'usr_profile', email: 'profile@example.com' })
const profile = {
  id: 'p_1', user_id: user.id, version: 3, nickname: '媽媽', name: '', gender: 'female',
  birth_date: '1958-05-12', birth_time: '06:00',
  birth_local_time: '06:00',
  birth_local_date: '1958-05-12',
  birth_place: '上海', birth_longitude: 121.4726, birth_timezone: 'Asia/Shanghai',
  birth_instant: '1958-05-11T22:00:00Z', note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

async function mockDetail(page: Page, onPatch: (route: Route) => unknown) {
  await mockWorkspace(page, { user })
  // 表单挂载会把旧档案的出生地匹配成经度（city_coords candidates 模式）；
  // 其余引擎工具显式失败，详情排盘走「未找到出生地座標」的降级路径。
  await page.route(/\/(engine|counsel)\/mcp$/, route => {
    const body = route.request().postDataJSON() as { params?: { name?: string } }
    if (body?.params?.name === 'city_coords') {
      return route.fulfill(json({
        name: '上海市', longitude: 121.4726, latitude: 31.2304, country: '上海市',
        candidates: [{
          name: '上海市', region: '上海市', country: '上海市', level: 'city',
          longitude: 121.4726, latitude: 31.2304, timezone: 'Asia/Shanghai',
        }],
      }))
    }
    return route.fulfill({ status: 502, json: { detail: 'engine unavailable in this journey' } })
  })
  await page.route('**/api/birth-profiles/p_1', route => (
    route.request().method() === 'PATCH' ? onPatch(route) : route.fulfill(json(profile))
  ))
}

async function startEditing(page: Page) {
  await page.goto('/app/birth-profiles/p_1')
  await expect(page.getByRole('heading', { name: '媽媽' })).toBeVisible()
  await page.getByRole('button', { name: '編輯' }).click()
  await page.getByLabel(/稱呼/).fill('媽媽（新）')
}

test('a conflicting save keeps the form and explains the conflict', async ({ page }) => {
  const patches: unknown[] = []
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
