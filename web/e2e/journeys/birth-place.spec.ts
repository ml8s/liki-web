// 出生地 → 经度闭环：多候选让用户选，选中的经度随档案保存。
import { test, expect } from '../fixtures'
import { json, mockWorkspace, testUser } from '../mocks'
import { fillBirthFields } from '../helpers'

const mcpOk = (data: unknown) => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    result: { content: [{ type: 'text', text: JSON.stringify(data) }] },
  }),
})

const savedProfile = {
  id: 'p_new', user_id: 'usr_test', version: 1, nickname: '测试', name: '', gender: 'female',
  birth_date: '1990-06-15', birth_time: '08:30',
  birth_local_time: '08:30',
  birth_local_date: '1990-06-15',
  birth_place: '湖南省长沙县', birth_longitude: 113.075, birth_timezone: 'Asia/Shanghai',
  birth_instant: '1990-06-14T23:30:00Z', note: '', third_party_consent: true,
  last_used_at: null, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

test('ambiguous birth place resolves to a persisted longitude', async ({ page }) => {
  await mockWorkspace(page, { user: testUser(), allowedSkills: ['liki'] })

  const created: Array<Record<string, unknown>> = []
  await page.route('**/api/birth-profiles', route => {
    if (route.request().method() !== 'POST') {
      return route.fulfill(json({ profiles: [], next_cursor: '' }))
    }
    created.push(route.request().postDataJSON() as Record<string, unknown>)
    return route.fulfill(json(savedProfile))
  })

  // city_coords 走 candidates 模式；其余引擎工具给空响应（详情页排盘降级即可）。
  await page.route(/\/(engine|counsel)\/mcp$/, route => {
    const body = route.request().postDataJSON() as { params?: { name?: string } }
    if (body?.params?.name === 'city_coords') {
      return route.fulfill(mcpOk({
        name: '长沙市', longitude: 113.0267, latitude: 28.1886, country: '湖南省',
        ambiguous: true,
        candidates: [
          { name: '长沙市', region: '湖南省', country: '湖南省', level: 'city', longitude: 113.0267, latitude: 28.1886, timezone: 'Asia/Shanghai' },
          { name: '长沙县', region: '湖南省', country: '湖南省', level: 'county', longitude: 113.075, latitude: 28.2502, timezone: 'Asia/Shanghai' },
        ],
      }))
    }
    return route.fulfill(mcpOk({}))
  })

  await page.goto('/app/birth-profiles/new')
  await page.getByLabel(/称呼|稱呼/).fill('测试')
  await fillBirthFields(page, page, {
    date: '1990-06-15',
    time: '08:30',
    gender: 'female',
    place: '长沙',
  })
  await expect(page.getByText(/找到多(个|個)同名地(点|點)/)).toBeVisible()
  await page.locator('.el-select__wrapper').nth(0).click()
  await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: '长沙县' }).last().click()
  await expect(page.locator('input[name="birth-longitude"]')).toHaveValue('113.0750')

  await page.locator('.el-checkbox').last().click()
  await page.getByRole('button', { name: /保存档案|儲存檔案/ }).click()

  await expect.poll(() => created.length).toBe(1)
  expect(created[0]).toMatchObject({
    nickname: '测试',
    birth_place: '湖南省长沙县',
    birth_longitude: 113.075,
    birth_timezone: 'Asia/Shanghai',
    birth_instant: '1990-06-14T23:30:00Z',
  })
})
