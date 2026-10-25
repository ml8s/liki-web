// v0 档案不猜测地点：显示待补充，用户重新选择后保存为 v2。
import { test, expect } from '../fixtures'
import { json, mcpOk, mockWorkspace, testUser } from '../mocks'

const legacy = {
  id: 'p_legacy', user_id: 'usr_test', version: 1, nickname: '旧档案', name: '',
  birth: {
    gender: 'female', date: '1990-06-15', time: '08:30',
    instant: '1990-06-14T23:30:00Z',
    location: { place: '上海', longitude: 121.4737, timezone: 'Asia/Shanghai', location_version: 0 },
  },
  note: '', third_party_consent: true,
  last_used_at: null, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const completed = {
  ...legacy,
  version: 2,
  birth: {
    ...legacy.birth,
    location: {
      place: '湖南省长沙市',
      place_id: 'CN-430100',
      location_version: 2,
      longitude: 112.9823,
      timezone: 'Asia/Shanghai',
    },
  },
}

test('v0 profile shows no guessed place and saves a v2 place', async ({ page }) => {
  await mockWorkspace(page, { user: testUser(), profiles: [legacy] })
  await page.route('**/api/me/sessions', route => route.fulfill(json([])))
  await page.route('**/api/me/auth/identities', route => route.fulfill(json([])))
  await page.route('**/api/client-errors', route => route.fulfill({ status: 204 }))
  await page.route('**/engine/mcp', route => {
    const body = route.request().postDataJSON() as { params?: { name?: string } }
    if (body?.params?.name === 'city_coords') return route.fulfill({ json: mcpOk({ candidates: [] }) })
    return route.fulfill({ status: 502, json: { detail: 'engine unavailable in this journey' } })
  })

  const dataRequests: string[] = []
  page.on('request', request => {
    if (request.url().includes('cities_data.json')) dataRequests.push(request.url())
  })

  let patchBody: Record<string, unknown> | null = null
  await page.route('**/api/birth-profiles/p_legacy', async route => {
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON() as Record<string, unknown>
      return route.fulfill(json(completed))
    }
    return route.fulfill(json(legacy))
  })

  await page.goto('/app/birth-profiles/p_legacy')
  await expect(page.getByText(/地點待補充|地点待补充|Place needed/).first()).toBeVisible()
  await expect(page.getByText('上海')).toHaveCount(0)

  await page.getByRole('button', { name: /編輯|编辑|Edit/ }).first().click()
  const place = page.locator('input[name="birth-place"]')
  await expect(place).toHaveValue('')

  await place.fill('changsha')
  const suggestions = page.locator('.el-autocomplete-suggestion__list li')
  await expect(suggestions.first()).toBeVisible()
  await expect(suggestions.filter({ hasText: /常山|Changshan/ })).toHaveCount(0)
  await suggestions.first().click()

  await page.getByRole('button', { name: /儲存檔案|保存档案|Save/ }).click()

  await expect.poll(() => patchBody).not.toBeNull()
  const body = patchBody as Record<string, unknown> | null
  const birth = body?.birth as Record<string, unknown> | undefined
  const location = birth?.location as Record<string, unknown> | undefined
  expect(location).toMatchObject({
    place_id: 'CN-430100',
    longitude: 112.9823,
    timezone: 'Asia/Shanghai',
  })
  expect(location?.place).toMatch(/长沙|長沙|Changsha/)
  expect(dataRequests.some(url => /cities_data\.json\?v=[0-9a-f]+/.test(url))).toBe(true)

  await expect(page.getByText(/地點待補充|地点待补充|Place needed/)).toHaveCount(0)
})
