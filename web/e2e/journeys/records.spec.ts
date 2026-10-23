// 工具记录（四术数历史）的标准回路：问卦自动落记录并在打开时回放快照；
// 择日等手动保存的记录走标准行操作（打开/重命名/删除，RecordActions）。
import { test, expect } from '../fixtures'
import type { Page } from '@playwright/test'
import { json, mockCityCoords, mockEngineTool, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })

const user = testUser({ id: 'usr_records', email: 'records@example.com' })

const liuyaoSnapshot = {
  name: '水火既济',
  ben_gua: '水火既济',
  gong: '坎',
  gong_wuxing: '水',
  lines: [],
  bian_yao: [],
  ri_chen_gan: '戊',
  ri_chen_zhi: '子',
  yue_jian_gan: '甲',
  yue_jian_zhi: '寅',
  xun_kong: ['子', '丑'],
  casting: { mode: 'auto', order: 'bottom_up', yaos: [7, 8, 8, 7, 8, 9], dong_yao: [6], casting_id: 'c1' },
  yong_shen: { name: '妻财', position: 6, wang_shuai: '相' },
  gua_ci: { name: '水火既济', gua_ci: '亨小，利贞', yao_ci: [] },
  wang_shuai: [],
  ri_chen_relations: [],
  patterns: [],
  timing_candidates: [],
}

function countEngineCalls(page: Page): { qigua: number; huangli: number } {
  const counts = { qigua: 0, huangli: 0 }
  page.on('requestfinished', request => {
    if (!/\/(engine|counsel)\/mcp$/.test(new URL(request.url()).pathname)) return
    const body = request.postData() ?? ''
    if (body.includes('"liuyao_qigua"')) counts.qigua += 1
    if (body.includes('"huangli_days"')) counts.huangli += 1
  })
  return counts
}

async function mockEngines(page: Page) {
  await mockEngineTool(page, 'liuyao_qigua', { casting: liuyaoSnapshot.casting })
  await mockEngineTool(page, 'liuyao_chart', liuyaoSnapshot)
  await mockCityCoords(page, { name: '上海市', region: '中國', longitude: 121.4726, latitude: 31.2304 })
  await mockEngineTool(page, 'huangli_days', [
    { date: '2026-10-01', suitability: 'recommended', jian_chu: '建', reason: '宜嫁娶' },
  ])
}

test('问卦自动保存记录，打开历史回放快照不再重摇', async ({ page }) => {
  await mockWorkspace(page, { user, sessions: [] })
  await mockEngines(page)
  const counts = countEngineCalls(page)
  const engineArguments: { tool: string; arguments: Record<string, unknown> }[] = []
  await page.route('**/engine/mcp', async route => {
    const body = JSON.parse(route.request().postData() ?? '{}') as { params?: { name?: string; arguments?: Record<string, unknown> } }
    if (body.params?.name) engineArguments.push({ tool: body.params.name, arguments: body.params.arguments ?? {} })
    await route.fallback()
  })

  const created: Record<string, unknown>[] = []
  await page.route('**/api/records**', route => {
    const request = route.request()
    if (request.method() === 'POST') {
      const body = request.postDataJSON() as Record<string, unknown>
      created.push(body)
      return route.fulfill(json({
        id: 'rec_1', kind: body.kind, title: body.title, input: body.input, output: body.output,
        created_at: '2026-10-18T00:00:00Z', updated_at: '2026-10-18T00:00:00Z',
      }))
    }
    return route.fulfill(json({ records: [] }))
  })

  await page.goto('/app/divination')
  await page.locator('textarea[placeholder*="項目"]').fill('這個項目能成嗎')
  await page.locator('input[name="casting-place"]').fill('上海')
  await page.locator('input[name="casting-place"]').blur()
  await expect(page.locator('.region-status')).toContainText('已匹配')
  await page.getByRole('button', { name: '手動填寫經度' }).click()
  await expect(page.locator('input[name="casting-longitude"]')).toHaveValue('121.4726')
  await page.getByRole('button', { name: '搖卦', exact: true }).click()
  await page.waitForSelector('.gua-name')

  // 摇卦即落一条记录：kind/input/output 与引擎快照一致。
  await expect.poll(() => created.length).toBe(1)
  const input = created[0].input as Record<string, unknown>
  expect(created[0].kind).toBe('liuyao')
  expect(input.question).toBe('這個項目能成嗎')
  expect(input.casting_mode).toBe('auto')
  expect(String(input.casting_instant)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/)
  expect(input.location).toMatchObject({
    place: '上海市',
    longitude: 121.4726,
    timezone: 'Asia/Shanghai',
    source: 'city',
  })
  expect((created[0].output as { name: string }).name).toBe('水火既济')
  expect(engineArguments.find(call => call.tool === 'liuyao_qigua')?.arguments).toEqual({ mode: 'auto' })
  expect(engineArguments.find(call => call.tool === 'tianwen_resolve')).toBeUndefined()
  const chartArguments = engineArguments.find(call => call.tool === 'liuyao_chart')?.arguments
  expect(chartArguments).toMatchObject({
    instant: input.casting_instant,
    timezone: 'Asia/Shanghai',
    longitude: 121.4726,
  })
  expect(chartArguments).toHaveProperty('yong_shen', '世爻')
  expect(chartArguments).not.toHaveProperty('casting_place')

  // 历史卡出现该记录。
  const historyCard = page.locator('.app-card', { hasText: '歷史記錄' })
  await expect(historyCard).toContainText('問卦 · 這個項目能成嗎')

  // 打开 = 回放快照：卦名仍在，且不重新摇卦。
  await historyCard.getByRole('button', { name: '打開' }).click()
  await expect(page.locator('.gua-name')).toHaveText('水火既济')
  expect(counts.qigua).toBe(1)
})

test('问卦缺少完整地点时不调用排盘引擎', async ({ page }) => {
  await mockWorkspace(page, { user, sessions: [] })
  await mockEngines(page)
  const counts = countEngineCalls(page)

  await page.goto('/app/divination')
  await page.locator('textarea[placeholder*="項目"]').fill('這個項目能成嗎')
  await page.getByRole('button', { name: '搖卦', exact: true }).click()

  await expect(page.getByText('請填寫起卦地點')).toBeVisible()
  expect(counts.qigua).toBe(0)
})

test('浏览器定位只预填可见表单，仍可手动修改', async ({ page }) => {
  await mockWorkspace(page, { user, sessions: [] })
  await page.context().grantPermissions(['geolocation'])
  await page.context().setGeolocation({ latitude: 31.2304, longitude: 121.4726 })

  await page.goto('/app/divination')
  await page.getByRole('button', { name: '使用當前定位' }).click()
  await expect(page.locator('input[name="casting-place"]')).toHaveValue('當前設備位置')
  await page.getByRole('button', { name: '手動填寫經度' }).click()
  await expect(page.locator('input[name="casting-longitude"]')).toHaveValue('121.4726')
  await expect(page.locator('input[name="casting-timezone"]')).toHaveValue(Intl.DateTimeFormat().resolvedOptions().timeZone)
  await expect(page.locator('.location-status')).toContainText('定位精度約')

  await page.locator('input[name="casting-place"]').fill('上海')
  await expect(page.locator('.location-status')).toContainText('瀏覽器定位僅用於填入可見表單')
})

test('择日记录行操作：打开重算、重命名、删除', async ({ page }) => {
  await mockWorkspace(page, { user, sessions: [] })
  await mockEngines(page)
  const counts = countEngineCalls(page)

  const list = [{
    id: 'rec_h1', user_id: user.id, kind: 'huangli', title: '擇日 · 婚禮 · 2026-10-01',
    input: { event: 'wedding', start_date: '2026-10-01', end_date: '2026-10-01' },
    created_at: '2026-10-17T00:00:00Z', updated_at: '2026-10-17T00:00:00Z',
  }]
  let patched: unknown = null
  let deleted = 0
  await page.route('**/api/records**', route => {
    const request = route.request()
    if (request.method() === 'PATCH') {
      patched = request.postDataJSON()
      return route.fulfill(json({ ...list[0], title: (request.postDataJSON() as { title: string }).title }))
    }
    if (request.method() === 'DELETE') {
      deleted += 1
      return route.fulfill(json({ status: 'deleted' }))
    }
    return route.fulfill(json({ records: list }))
  })

  await page.goto('/app/huangli')
  const historyCard = page.locator('.app-card', { hasText: '歷史記錄' })
  await expect(historyCard).toContainText('擇日 · 婚禮 · 2026-10-01')

  // 打开：按存下来的输入重算（引擎被再次调用）。
  await historyCard.getByRole('button', { name: '打開' }).click()
  await expect.poll(() => counts.huangli).toBe(1)

  // 重命名走标准 prompt。
  await historyCard.getByRole('button', { name: '重新命名' }).click()
  const prompt = page.locator('.el-message-box')
  await prompt.locator('.el-message-box__input input').fill('十月婚禮擇日')
  await prompt.getByRole('button', { name: '儲存' }).click()
  await expect(historyCard).toContainText('十月婚禮擇日')
  expect(patched).toEqual({ title: '十月婚禮擇日' })

  // 删除走标准 confirm，行消失。
  await historyCard.getByRole('button', { name: '刪除' }).click()
  const confirm = page.locator('.el-message-box').filter({ hasText: '十月婚禮擇日' })
  await confirm.getByRole('button', { name: '刪除' }).click()
  await expect(historyCard).toBeHidden()
  expect(deleted).toBe(1)
})
