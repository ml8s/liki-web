// 工具记录（四术数历史）的标准回路：问卦自动落记录并在打开时回放快照；
// 择日等手动保存的记录走标准行操作（打开/重命名/删除，RecordActions）。
import { test, expect } from '../fixtures'
import type { Page } from '@playwright/test'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })

const user = testUser({ id: 'usr_records', email: 'records@example.com' })

const liuyaoSnapshot = {
  board: { name: '水火既济', gong: '坎', lines: [], xun_kong: ['子', '丑'] },
  casting: { dong_yao: [6] },
  focus: { yong_shen: '妻财', world: '世', other: '应' },
  patterns: [],
  yong_shen: { name: '妻财', position: '三爻' },
  gua_ci: { gua: '既济', ci: '亨小，利贞' },
  timing_candidates: [],
}

async function mockEngines(page: Page, counts: { qigua: number; huangli: number }) {
  await page.route('**/api/engine/tools/liuyao_qigua', route => {
    counts.qigua += 1
    return route.fulfill(json({ casting: liuyaoSnapshot.casting }))
  })
  await page.route('**/api/engine/tools/liuyao_chart', route => route.fulfill(json(liuyaoSnapshot)))
  await page.route('**/api/engine/tools/tianwen_time', route => route.fulfill(json({
    solar: '2026-10-18T12:00:00+08:00',
    lunar: { year: 2026, month: 9, day: 8 },
  })))
  await page.route('**/api/engine/tools/huangli_days', route => {
    counts.huangli += 1
    return route.fulfill(json([
      { date: '2026-10-01', suitability: 'recommended', jian_chu: '建', reason: '宜嫁娶' },
    ]))
  })
}

test('问卦自动保存记录，打开历史回放快照不再重摇', async ({ page }) => {
  const counts = { qigua: 0, huangli: 0 }
  await mockWorkspace(page, { user, sessions: [] })
  await mockEngines(page, counts)

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
  await page.locator('input[placeholder*="項目"]').fill('這個項目能成嗎')
  await page.click('button:has-text("搖卦")')
  await page.waitForSelector('.gua-name')

  // 摇卦即落一条记录：kind/input/output 与引擎快照一致。
  await expect.poll(() => created.length).toBe(1)
  expect(created[0].kind).toBe('liuyao')
  expect(created[0].input).toEqual({ question: '這個項目能成嗎' })
  expect((created[0].output as { board: { name: string } }).board.name).toBe('水火既济')

  // 历史卡出现该记录。
  const historyCard = page.locator('.app-card', { hasText: '歷史記錄' })
  await expect(historyCard).toContainText('問卦 · 這個項目能成嗎')

  // 打开 = 回放快照：卦名仍在，且不重新摇卦。
  await historyCard.getByRole('button', { name: '打開' }).click()
  await expect(page.locator('.gua-name')).toHaveText('水火既济')
  expect(counts.qigua).toBe(1)
})

test('择日记录行操作：打开重算、重命名、删除', async ({ page }) => {
  const counts = { qigua: 0, huangli: 0 }
  await mockWorkspace(page, { user, sessions: [] })
  await mockEngines(page, counts)

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
  const confirm = page.locator('.el-message-box')
  await confirm.getByRole('button', { name: '刪除' }).click()
  await expect(historyCard).toBeHidden()
  expect(deleted).toBe(1)
})
