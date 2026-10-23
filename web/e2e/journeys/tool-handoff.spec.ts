import { expect, test } from '../fixtures'
import type { Locator, Page } from '@playwright/test'
import { mockEngineTool, mockPageview } from '../mocks'
import { baziChart, bond } from '../engine-fixtures'
import { fillBirthFields } from '../helpers'

async function clickWithoutNavigation(target: Locator) {
  await target.evaluate(element => element.removeAttribute('href'))
  await target.click()
}

function consultationHandoff(page: Page) {
  return page.evaluate(() => {
    const raw = sessionStorage.getItem('liki:form-drafts')
    if (!raw) return null
    const drafts = JSON.parse(raw) as Record<string, { data?: unknown }>
    return drafts['handoff:consultation']?.data ?? null
  })
}

test.describe('public tool consultation handoff', () => {
  test.beforeEach(async ({ page }) => {
    await mockPageview(page)
  })

  test('He Pan preserves both people without putting facts in the URL', async ({ page }) => {
    await mockEngineTool(page, 'bazi_chart', baziChart)
    await mockEngineTool(page, 'bazi_bond', bond)
    await page.goto('/zh-Hant/hepan/')

    const people = page.locator('#hepan-tool fieldset')
    for (const person of [people.first(), people.nth(1)]) {
      await fillBirthFields(page, person, {
        date: '1990-05-12',
        time: '08:30',
        gender: 'female',
        place: '长沙',
      })
      await person.locator('input[name$="-timezone"]').fill('Asia/Shanghai')
      await person.locator('input[name$="-longitude"]').fill('112.9823')
    }
    await page.locator('#hepan-tool .el-checkbox').click()
    await page.locator('#hepan-tool button[type="submit"]').click()
    await expect(page.locator('#hepan-tool .tool-cta')).toBeVisible()
    await clickWithoutNavigation(page.locator('#hepan-tool .tool-cta'))

    const context = await consultationHandoff(page)
    expect(context).toMatchObject({
      version: 1,
      scenario: 'hepan',
      second_birth: {
        date: '1990-05-12',
        time: '08:30',
        location: { longitude: 112.9823, timezone: 'Asia/Shanghai' },
      },
      third_party_consent: true,
    })
    const birth = await page.evaluate(() => sessionStorage.getItem('liki:form-drafts'))
    expect(birth).toContain('"handoff:birth"')
    expect(page.url()).not.toContain('1990')
  })

  test('divination preserves the non-reproducible casting receipt', async ({ page }) => {
    const casting = {
      mode: 'auto',
      order: 'bottom_up',
      yaos: [7, 8, 8, 7, 8, 9],
      dong_yao: [6],
      casting_id: 'cast_handoff_1',
    }
    await mockEngineTool(page, 'liuyao_qigua', { casting })
    await page.goto('/zh-Hant/divination/')

    await page.locator('#divination-tool textarea').fill('這個項目能成嗎')
    await page.locator('#divination-tool button[type="submit"]').click()
    await expect(page.locator('#divination-tool .gua-name')).toBeVisible()
    await clickWithoutNavigation(page.locator('#divination-tool .tool-cta'))

    await expect.poll(() => consultationHandoff(page)).toMatchObject({
      version: 1,
      scenario: 'divination',
      question: '這個項目能成嗎',
      casting: { casting_id: 'cast_handoff_1' },
    })
  })

  test('Huangli preserves the selected event and date range', async ({ page }) => {
    await mockEngineTool(page, 'huangli_days', [])
    await page.goto('/zh-Hant/huangli/')
    await page.locator('#huangli-tool input[name="huangli-start-date"]').fill('2026-10-01')
    await page.locator('#huangli-tool input[name="huangli-end-date"]').fill('2026-10-03')
    await page.locator('#huangli-tool button[type="submit"]').click()
    await expect(page.locator('#huangli-tool .tool-cta')).toBeVisible()
    await clickWithoutNavigation(page.locator('#huangli-tool .tool-cta'))

    await expect.poll(() => consultationHandoff(page)).toEqual({
      version: 1,
      scenario: 'huangli',
      event: 'wedding',
      start_date: '2026-10-01',
      end_date: '2026-10-03',
    })
  })

  test('Feng Shui preserves birth and layout facts', async ({ page }) => {
    await mockEngineTool(page, 'bazhai_chart', {
      ming_gua: { gua: { name: '艮' }, group: '西四命' },
      ba_zhai_dirs: {},
    })
    await mockEngineTool(page, 'xuankong_chart', {
      yun: { year: 2026, yun_number: 9, yun_name: '九运', start_year: 2024, end_year: 2043 },
      gong_wei: [],
      four_situation: {},
      zuo_shan: 0,
      zuo_shan_name: '子',
      xiang_shan: 12,
      xiang_shan_name: '午',
      shou_shan_chu_sha: {},
    })
    await page.goto('/zh-Hant/fengshui/')
    await page.locator('#fengshui-tool input[name="fengshui-birth-year"]').fill('1990')
    await page.locator('#fengshui-tool .el-radio-button').nth(1).click()
    await page.locator('#fengshui-tool button[type="submit"]').click()
    await expect(page.locator('#fengshui-tool .tool-cta')).toBeVisible()
    await clickWithoutNavigation(page.locator('#fengshui-tool .tool-cta'))

    await expect.poll(() => consultationHandoff(page)).toMatchObject({
      version: 1,
      scenario: 'fengshui',
      birth_year: 1990,
      gender: 'male',
      layout: { zuo_shan: 0 },
    })
  })
})
