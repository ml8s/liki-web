// @ts-check
// 引擎契约回归：用「真实引擎输出形状」渲染免费排盘与合盘，防止字段漂移被假 mock 掩盖。
// 排盘/合盘 = 前端 TS 直连引擎 RPC（只读），契约归 liki/engine。
import { test, expect } from '@playwright/test'
import { RPC_METHODS } from '../../app/src/api/engine'

test.use({ locale: 'zh-Hant' })

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })
const rpc = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ result: { data } }) })

const user = { id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant' }
const profile = {
  id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', gender: 'female', relation: '家人',
  calendar: 'solar', birth_date: '1958-05-12', birth_time: '06:00', birth_time_known: 'known',
  birth_place: '上海', note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

// ── 真实引擎输出形状（与 liki/engine Go 结构对齐，勿简化为字符串）──
const baziChart = {
  nian: { gan: '戊', zhi: '戌', na_yin: '平地木' },
  yue: { gan: '丁', zhi: '巳', na_yin: '沙中土' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1958,
  gender: 'female',
  da_yun: {
    direction: '逆排', current_step_index: 5, next_step_in_years: 2,
    steps: [
      { gan: '丙', zhi: '辰', name: '丙辰', wuxing: '火', shi_shen: '偏财', start_year: 1968, end_year: 1977, start_date: '1968-01-01', end_date: '1977-12-31' },
      { gan: '辛', zhi: '亥', name: '辛亥', wuxing: '金', shi_shen: '正印', start_year: 2018, end_year: 2027, start_date: '2018-01-01', end_date: '2027-12-31' },
    ],
  },
}
const baziFull = {
  nian: { gan: '戊', zhi: '戌', na_yin: '平地木', shi_shens: [{ shi_shen: '七杀', name: '七杀' }], cang_gan: { main: '戊', mid: '辛', minor: '丁' }, shen_sha: [{ name: '天乙贵人' }, { name: '华盖' }], is_void: false },
  yue: { gan: '丁', zhi: '巳', na_yin: '沙中土', shi_shens: [{ shi_shen: '正财', name: '正财' }], cang_gan: { main: '丙', mid: '戊', minor: '庚' }, shen_sha: [{ name: '驿马' }], is_void: false },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金', shi_shens: [{ shi_shen: '日主', name: '日主' }], cang_gan: { main: '庚', mid: '壬', minor: '戊' }, shen_sha: [{ name: '驿马' }], is_void: false },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金', shi_shens: [{ shi_shen: '劫财', name: '劫财' }], cang_gan: { main: '乙' }, shen_sha: [{ name: '羊刃' }], is_void: true },
  san_yuan: { tai_yuan: { gan: '壬', zhi: '申' }, ming_gong: { gan: '癸', zhi: '亥' }, shen_gong: { gan: '乙', zhi: '卯' } },
  yong_shen: {
    fu_yi: { qiangruo: '身弱', yong: '金', xi: '金水', ji: '火土' },
    tiao_hou: { yong: '辛', xi: '壬', season: '夏' },
    ge_ju: { ge_ju: '正财格', yong_fa: '财旺身弱宜比劫帮身', yong: '辛', xi: '金水', ji: '火土' },
  },
}
const ziweiChart = { school: 'x', gong_wei: [], ming_gong: 0, shen_gong: 5, ju_shu: 0, ju_shu_name: '水二局', ziwei_pos: 0, si_hua: {}, palace_facts: [], nian_gan: '戊', shi_zhi: '卯', birth_year: 1958, lunar_month: 4 }
const ziweiFull = {
  gong_wei: [
    { index: 0, name: '命宫', gan: '癸', zhi: '亥', is_shen_gong: false, xing_yao: [{ xing: 0, name: '紫微', is_zhu_xing: true, liang_du: '庙' }, { xing: 1, name: '贪狼', is_zhu_xing: false, liang_du: '平' }] },
    { index: 11, name: '父母', gan: '甲', zhi: '寅', is_shen_gong: true, xing_yao: [{ xing: 7, name: '破军', is_zhu_xing: true, liang_du: '陷' }] },
    { index: 1, name: '兄弟', gan: '壬', zhi: '子', is_shen_gong: false, xing_yao: [] },
    { index: 2, name: '夫妻', gan: '壬', zhi: '丑', is_shen_gong: false, xing_yao: [] },
    { index: 3, name: '子女', gan: '辛', zhi: '寅', is_shen_gong: false, xing_yao: [] },
    { index: 4, name: '财帛', gan: '辛', zhi: '卯', is_shen_gong: false, xing_yao: [] },
    { index: 5, name: '疾厄', gan: '庚', zhi: '辰', is_shen_gong: false, xing_yao: [] },
    { index: 6, name: '迁移', gan: '庚', zhi: '巳', is_shen_gong: false, xing_yao: [] },
    { index: 7, name: '仆役', gan: '己', zhi: '午', is_shen_gong: false, xing_yao: [] },
    { index: 8, name: '官禄', gan: '己', zhi: '未', is_shen_gong: false, xing_yao: [] },
    { index: 9, name: '田宅', gan: '戊', zhi: '申', is_shen_gong: false, xing_yao: [] },
    { index: 10, name: '福德', gan: '戊', zhi: '酉', is_shen_gong: false, xing_yao: [] },
  ],
  ming_gong: 0, shen_gong: 11, ju_shu: 0, ju_shu_name: '水二局', ziwei_pos: 0, si_hua: {}, palace_facts: [],
}
const bazhai = {
  ming_gua: { gua: { index: 7, name: '艮', wuxing: '土', yin_yang: '阳' }, group: '西四命', year_boundary: 'gregorian_calendar_year' },
  ba_zhai_dirs: {
    sheng_qi: ['西南'], tian_yi: ['正西'], yan_nian: ['西北'], fu_wei: ['正东'],
    huo_hai: ['正北'], wu_gui: ['东北'], liu_sha: ['正南'], jue_ming: ['东南'],
  },
  liu_nian_xing: {
    year: 2026, year_boundary: 'gregorian_calendar_year', ru_zhong: '三碧禄存',
    gong_wei: [
      { gong_num: 1, xing: 4, xing_name: '四绿文曲', wuxing: '木', ru_zhong: false },
      { gong_num: 2, xing: 5, xing_name: '五黄廉贞', wuxing: '土', ru_zhong: false },
      { gong_num: 3, xing: 6, xing_name: '六白武曲', wuxing: '金', ru_zhong: false },
      { gong_num: 4, xing: 7, xing_name: '七赤破军', wuxing: '金', ru_zhong: false },
      { gong_num: 5, xing: 3, xing_name: '三碧禄存', wuxing: '木', ru_zhong: true },
      { gong_num: 6, xing: 8, xing_name: '八白左辅', wuxing: '土', ru_zhong: false },
      { gong_num: 7, xing: 9, xing_name: '九紫右弼', wuxing: '火', ru_zhong: false },
      { gong_num: 8, xing: 1, xing_name: '一白贪狼', wuxing: '水', ru_zhong: false },
      { gong_num: 9, xing: 2, xing_name: '二黑巨门', wuxing: '土', ru_zhong: false },
    ],
  },
}
const bond = {
  zhu_cross: {
    pairs: [
      { jia_zhu: 'ri', yi_zhu: 'ri', jia_gan: '壬', yi_gan: '戊', jia_zhi: '申', yi_zhi: '午', gan_guan_xi: { gan_a: '壬', gan_b: '戊', type: '相克', relation: '壬戊相克' }, zhi_guan_xi: { zhi_a: '申', zhi_b: '午', type: '三合', detail: '三合' } },
      { jia_zhu: 'nian', yi_zhu: 'nian', jia_gan: '戊', yi_gan: '甲', jia_zhi: '戌', yi_zhi: '子', gan_guan_xi: { gan_a: '戊', gan_b: '甲', type: '相生', relation: '戊甲相生' }, zhi_guan_xi: { zhi_a: '戌', zhi_b: '子', type: '六合', detail: '六合' } },
      { jia_zhu: 'yue', yi_zhu: 'shi', jia_gan: '丁', yi_gan: '壬', jia_zhi: '巳', yi_zhi: '寅', gan_guan_xi: { gan_a: '丁', gan_b: '壬', type: '天干五合', relation: '丁壬合' }, zhi_guan_xi: { zhi_a: '巳', zhi_b: '寅', type: '六冲', detail: '六冲' } },
    ],
  },
  shi_shen_cross: { jia_dui_yi: { nian_gan: '偏财', ri_gan: '日主' }, yi_dui_jia: { nian_gan: '偏印', ri_gan: '日主' } },
}

const qimingPick = {
  wuxing1: '金', wuxing2: '水',
  pools: [
    { slot: 'first', chars: [{ char: '锐', frequency: '2k' }, { char: '铭', frequency: '1k' }] },
    { slot: 'second', chars: [{ char: '泓', frequency: '1k' }] },
  ],
}
const liuyaoSnapshot = {
  question: '这个项目能成吗',
  casting: { mode: 'auto', rounds: 3, yaos: [[1, 2, 3], [4, 5, 6], [7, 8, 9]], dong_yao: [3], casting_id: 'c1', order: 0 },
  board: { name: '水火既济', gong: '坎', gong_wuxing: '水', xun_kong: ['午', '未'], lines: [], bian_lines: [], ri_chen_gan: '戊', ri_chen_zhi: '子', yue_jian_gan: '甲', yue_jian_zhi: '寅' },
  focus: { yong_shen: '妻财', yong_line: '三爻', world: '三爻', other: '六爻' },
  timing_candidates: [{ note: '午月' }],
}
const huangliDays = {
  event: 'wedding', event_label: '结婚', range: { start_date: '2026-10-01', end_date: '2026-10-31', days: 31 },
  recommended: [{ date: '2026-10-06', jian_chu: '除', suitability: 'recommended', reason: '宜嫁娶' }],
  unsuitable: [{ date: '2026-10-09', suitability: 'unsuitable', reason: '诸事不宜' }],
}

function routeEngine(page) {
  return page.route('**/jsonrpc', async route => {
    const { method } = await route.request().postDataJSON()
    let data
    switch (method) {
      case RPC_METHODS.CityCoords: data = { longitude: 121.47, latitude: 31.23 }; break
      case RPC_METHODS.TianwenTime: data = { solar: '1958-05-12T06:00:00+08:00', lunar: { year: 1958, month: 4, day: 24 } }; break
      case RPC_METHODS.BaziChart: data = baziChart; break
      case RPC_METHODS.BaziFullChart: data = baziFull; break
      case RPC_METHODS.ZiweiChart: data = ziweiChart; break
      case RPC_METHODS.ZiweiFullChart: data = ziweiFull; break
      case RPC_METHODS.BazhaiChart: data = bazhai; break
      case RPC_METHODS.BaziBond: data = bond; break
      case RPC_METHODS.QimingPick: data = qimingPick; break
      case RPC_METHODS.LiuyaoQigua: data = { casting: liuyaoSnapshot.casting }; break
      case RPC_METHODS.LiuyaoChart: data = liuyaoSnapshot; break
      case RPC_METHODS.HuangliDays: data = huangliDays; break
      default: data = {}
    }
    return route.fulfill(rpc(data))
  })
}

function routeApp(page) {
  page.route('**/api/me', r => r.fulfill(json(user)))
  page.route('**/api/me/entitlements', r => r.fulfill(json({ entitlements: [], allowed_skills: ['liki'] })))
}

test('免费排盘按真实引擎契约渲染（八字/紫微/八宅）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/profiles**', r => (r.request().url().endsWith('/profiles/p_1') ? r.fulfill(json(profile)) : r.fulfill(json({ profiles: [profile], next_cursor: '' }))))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/profiles/p_1')
  await page.waitForSelector('.pillars', { timeout: 12000 }) // 自动排盘
  await expect(page.locator('.pillar .gl').first()).toContainText('戊戌')
  await expect(page.locator('.pillar .cang').first()).toContainText('戊 辛 丁') // 藏干
  await expect(page.locator('.pillar .sub').first()).toContainText('七杀') // 十神（shi_shens 数组首项）
  await expect(page.locator('.void-tag')).toHaveCount(1) // 旬空
  await expect(page.locator('.ss-chip')).toContainText(['天乙贵人', '华盖']) // 神煞
  await expect(page.locator('.san-yuan .sy-body')).toContainText('胎元 壬申')

  await page.locator('.el-tabs__item:has-text("紫微")').click()
  await expect(page.locator('.palace .pstar').first()).toContainText('紫微庙') // 庙旺平陷
  await expect(page.locator('.palace.shen .pname')).toHaveText('父母') // 身宫

  await page.locator('.el-tabs__item:has-text("八宅")').click()
  await expect(page.locator('.mingua')).toHaveText('艮')
  await expect(page.locator('.fly-cell')).toHaveCount(9) // 流年飞星九宫
  await expect(page.locator('.ln-row')).toContainText('三碧禄存')
})

test('合盘按真实引擎契约渲染（bazi.bond）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/profiles**', r => (r.request().url().endsWith('/profiles/p_1') ? r.fulfill(json(profile)) : r.fulfill(json({ profiles: [profile], next_cursor: '' }))))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/synastry')
  await page.waitForSelector('.b-form', { timeout: 10000 })
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("妈妈")').first().click()
  await page.waitForTimeout(300)
  await page.locator('.b-form .el-radio-button:has-text("男")').click()
  await page.getByRole('combobox', { name: '出生日期' }).click()
  await page.locator('.el-picker-panel .el-date-table td.available').first().click()
  await page.getByRole('combobox', { name: '出生時間' }).fill('08:00')
  await page.locator('.b-form button[type="submit"]').click()
  await page.waitForSelector('.rel-sec', { timeout: 12000 })
  await expect(page.locator('.rel-text')).toContainText('壬申') // 日主
  await expect(page.locator('.rel-text')).toContainText('三合') // 地支关系（对象 detail）
  await expect(page.locator('.rel-chip')).toContainText(['六合', '六冲'])
  await expect(page.locator('.rel-chip')).toContainText(['妈妈年柱看對方：偏财'])
})

test('问卦按真实契约渲染（liuyao_snapshot）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/profiles**', r => r.fulfill(json({ profiles: [profile], next_cursor: '' })))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/divination')
  await page.waitForSelector('input[placeholder*="項目"]', { timeout: 10000 })
  await page.locator('input[placeholder*="項目"]').fill('這個項目能成嗎')
  await page.click('button:has-text("搖卦")')
  await page.waitForSelector('.gua-name', { timeout: 12000 })
  await expect(page.locator('.gua-name')).toHaveText('水火既济') // board.name
  await expect(page.locator('.gua-info')).toContainText('坎') // board.gong
  await expect(page.locator('.rel-text')).toContainText('妻财') // focus.yong_shen
})

test('择日按真实契约渲染（huangli_days）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/profiles**', r => r.fulfill(json({ profiles: [profile], next_cursor: '' })))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/huangli')
  await page.waitForSelector('.el-date-editor', { timeout: 10000 })
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("結婚")').first().click()
  await page.getByRole('combobox', { name: '起始日期' }).fill('2026-10-01')
  await page.keyboard.press('Enter')
  await page.getByRole('combobox', { name: '結束日期' }).fill('2026-10-31')
  await page.keyboard.press('Enter')
  await page.locator('button[type="submit"]').click()
  await page.waitForSelector('.day.good', { timeout: 12000 })
  await expect(page.locator('.day.good .d-date').first()).toHaveText('2026-10-06') // recommended[].date
  await expect(page.locator('.day.bad')).toHaveCount(1) // unsuitable
})
