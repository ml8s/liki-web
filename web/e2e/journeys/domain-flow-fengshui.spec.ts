import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'
import { mockEngineTool } from '../mocks'

const bazhai = {
  ming_gua: { gua: { name: '艮', wuxing: '土' }, group: '西四命' },
  ba_zhai_dirs: {
    sheng_qi: '西南', tian_yi: '西北', yan_nian: '西', fu_wei: '东北',
    huo_hai: '南', wu_gui: '北', liu_sha: '东', jue_ming: '东南',
  },
}

const xuankong = {
  yun: { year: 2026, yuan: '下元', yun_number: 9, yun_name: '九运', start_year: 2024, end_year: 2043 },
  gong_wei: Array.from({ length: 9 }, (_, index) => ({
    gong_num: index + 1,
    yun_xing: { number: 5, name: '五黄廉贞', wuxing: '土' },
    shan_xing: { number: 9, name: '九紫右弼', wuxing: '火' },
    xiang_xing: { number: 9, name: '九紫右弼', wuxing: '火' },
  })),
  four_situation: { name: '双星会坐', sit_mountain_timely: true, face_facing_timely: false, sit_facing_timely: true, face_mountain_timely: false },
  fu_yin_layers: [],
  fan_yin_layers: [],
  xing_jia_hui: [],
  zuo_shan: 0,
  zuo_shan_name: '子',
  xiang_shan: 12,
  xiang_shan_name: '午',
  shou_shan_chu_sha: { zheng_shen: 9, ling_shen: 1, shou_shan: true, chu_sha: false },
}

const layout = {
  group: '西四命',
  ming_gua: '艮',
  door: { direction: '西南', gua_name: '坤', wuxing: '土', youxing: '生气', rating: '大吉', group: '西四卦', match: '吉' },
  master: { direction: '西', gua_name: '兑', wuxing: '金', youxing: '延年', rating: '吉', group: '西四卦', match: '吉' },
  stove: { direction: '西北', gua_name: '乾', wuxing: '金', youxing: '天医', rating: '吉', group: '西四卦', match: '吉' },
}

test('prefills the fengshui layout from a marketing handoff', async ({ page }) => {
  await mockFlowWorkspace(page, { profiles: [flowProfile('person-1', '张三')] })
  // The prefilled layout is reactive: once the profile is selected it computes.
  await mockEngineTool(page, 'bazhai_chart', bazhai)
  await mockEngineTool(page, 'xuankong_chart', xuankong)
  await mockEngineTool(page, 'bazhai_layout', layout)

  await page.addInitScript(context => {
    try {
      const key = 'liki:form-drafts'
      const raw = sessionStorage.getItem(key)
      const drafts = raw ? JSON.parse(raw) as Record<string, unknown> : {}
      drafts['handoff:consultation'] = { data: context, createdAt: Date.now() }
      sessionStorage.setItem(key, JSON.stringify(drafts))
    } catch {
      // sessionStorage unavailable: the test's navigation below will fail visibly.
    }
  }, {
    version: 1,
    scenario: 'fengshui',
    birth_year: 1990,
    gender: 'female',
    layout: {
      zuo_shan: 0,
      period_date: '2026-01-01',
      door_gua: '坤',
      master_gua: '兑',
      stove_gua: '乾',
    },
  })

  await page.goto('/app/?service=fengshui&locale=zh-Hans')
  const form = page.locator('.flow-fengshui-form')
  await expect(form).toBeVisible()
  await expect(form.getByText('门 · 坤')).toBeVisible()
  await expect(form.getByText('主卧 · 兑')).toBeVisible()
  await expect(form.getByText('灶 · 乾')).toBeVisible()
})

test('computes the fengshui focused form and renders all deterministic artifacts', async ({ page }) => {
  await mockFlowWorkspace(page, { profiles: [flowProfile('person-1', '张三')] })
  await mockEngineTool(page, 'bazhai_chart', bazhai)
  await mockEngineTool(page, 'xuankong_chart', xuankong)
  await mockEngineTool(page, 'bazhai_layout', layout)

  await page.goto('/app/?domain=fengshui&seed=fengshui')
  const form = page.locator('.flow-fengshui-form')
  await expect(form).toBeVisible()

  const pickGua = async (label: string, gua: string) => {
    const input = form.getByRole('combobox', { name: label })
    await input.click({ force: true })
    const listbox = page.locator(`[id="${await input.getAttribute('aria-controls')}"]`)
    await listbox.getByRole('option', { name: `${label} · ${gua}` }).click()
    await expect(listbox).toBeHidden()
  }
  await pickGua('门', '坤')
  await pickGua('主卧', '兑')
  await pickGua('灶', '乾')

  await form.getByRole('button', { name: '风水' }).click()
  await expect(page.locator('.artifact-capsule')).toBeVisible()

  await page.locator('.artifact-capsule').click()
  await expect(page.locator('.bazhai-chart-panel')).toBeVisible()
  await expect(page.locator('.xuankong-panel')).toBeVisible()
  await expect(page.locator('.bazhai-layout-panel')).toBeVisible()

  await page.getByRole('button', { name: '保存记录' }).click()
  await expect(page.locator('.artifact-capsule')).toBeVisible()
})
