import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import type { BaziChartData, BaziFuYi, BaziFullChartData } from '@shared/api/engine/bazi'

// Provenance fixtures: web/shared/components/charts/BaziPanel.vue,
// web/shared/components/charts/bazi/{BaziPillars,BaziRelations,BaziCycles}.vue,
// and web/e2e/engine-fixtures.ts (baziFull).
const baziChart: BaziChartData = {
  nian: { gan: '庚', zhi: '午', na_yin: '路旁土' },
  yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1990,
  gender: 'female',
  zi_shi_rule: '晚子时属本日',
  da_yun: {
    start_date: '1996-02-04',
    start_year_after: 6,
    start_month_after: 7,
    start_day_after: 12,
    direction: '顺排',
    steps: [
      { gan: '壬', zhi: '午', name: '壬午', wuxing: '木', shi_shen: '比肩', start_date: '1996-02-04', end_date: '2005-02-03', start_year: 1996, end_year: 2005 },
      { gan: '癸', zhi: '未', name: '癸未', wuxing: '木', shi_shen: '劫财', start_date: '2006-02-04', end_date: '2015-02-03', start_year: 2006, end_year: 2015 },
      { gan: '甲', zhi: '申', name: '甲申', wuxing: '木', shi_shen: '食神', start_date: '2016-02-04', end_date: '2025-02-03', start_year: 2016, end_year: 2025 },
    ],
  },
}

const sourceBackedFuYi: BaziFuYi = { qiangruo: '身旺', yong: '木', xi: '水', ji: '土' }

const sourceBackedBaziFull: BaziFullChartData = {
  ...baziChart,
  nian: { ...baziChart.nian, cang_gan: { main: '丁', mid: '己' }, shi_shens: [{ shi_shen: '正官' }], shen_sha: [{ name: '天乙贵人' }], is_void: false },
  yue: { ...baziChart.yue, cang_gan: { main: '丙', mid: '庚', minor: '戊' }, shi_shens: [{ shi_shen: '正印' }], shen_sha: [{ name: '驿马' }], is_void: false },
  ri: { ...baziChart.ri, cang_gan: { main: '庚', mid: '壬', minor: '戊' }, shi_shens: [{ shi_shen: '日主' }], shen_sha: [], is_void: false },
  shi: { ...baziChart.shi, cang_gan: { main: '乙' }, shi_shens: [{ shi_shen: '伤官' }], shen_sha: [{ name: '羊刃' }], is_void: true },
  san_yuan: { tai_yuan: { gan: '壬', zhi: '午' }, ming_gong: { gan: '癸', zhi: '未' }, shen_gong: { gan: '甲', zhi: '申' } },
  fu_yi: sourceBackedFuYi,
  tiao_hou: { primary_wuxing: '木', secondary_wuxing: '水' },
  ge_ju: { ge_ju: '正财格', yong_fa: '顺用' },
}

const relationFixture: BaziFullChartData = {
  ...sourceBackedBaziFull,
  gan_he: [{ gan_a: '庚', gan_b: '辛', pillar_a: 0, pillar_b: 1, he_element: '金' }],
  ten_god_states: [{
    shi_shen: '正官', wuxing: '土', transparent: true, hidden: false, rooted: true, timely: true, count: 2, strength: 'strong',
  }],
  element_states: [{
    wuxing: '金', season: '旺', season_strength: '旺', strength: 'strong', transparent: true, rooted: true,
    controls: '木', controlled_by: '火', controller_strength: 'strong', reasons: [],
  }],
  fu_yi: { ...sourceBackedFuYi, wuxing_count: { 金: 3 } },
}

describe('BaziFullChartPanel | web/shared/components/charts/BaziPanel.vue + bazi source family', () => {
  it('preserves full-chart section order: pillars, relations/tables, cycles, yongshen', () => {
    const wrapper = mount(BaziFullChartPanel, { props: { result: relationFixture } })
    const orderedClasses = [
      'pillars', 'rel-table', 'analysis-grid', 'dayun', 'ys',
    ]
    const rootClasses = [...(wrapper.element as HTMLElement).children].map(
      element => element.className.split(' ').filter(name => orderedClasses.includes(name))[0] ?? '',
    ).filter(Boolean)

    expect(wrapper.find('.pillars').exists()).toBe(true)
    expect(wrapper.find('.rel-table').exists()).toBe(true)
    expect(wrapper.find('.analysis-grid').exists()).toBe(true)
    expect(wrapper.find('.dayun').exists()).toBe(true)
    expect(wrapper.find('.ys').exists()).toBe(true)
    expect(rootClasses).toEqual(orderedClasses)
    expect(wrapper.find('.analysis-grid table')).toBeTruthy()
  })

  it('renders the source-backed e2e fixture without optional relation fields', () => {
    const wrapper = mount(BaziFullChartPanel, { props: { result: sourceBackedBaziFull } })

    expect(wrapper.findAll('.pillar')).toHaveLength(4)
    expect(wrapper.find('.rel-table').exists()).toBe(false)
    expect(wrapper.find('.analysis-grid').exists()).toBe(false)
    expect(wrapper.find('.shensha').text()).toContain('天乙贵人')
    expect(wrapper.find('.san-yuan').text()).toContain('壬午')
    expect(wrapper.findAll('.dy')).toHaveLength(3)
    expect(wrapper.find('.ys').text()).toContain('身旺')
  })

  it('renders a required-fields-only result through every absent optional branch', () => {
    const result: BaziFullChartData = {
      nian: { gan: '庚', zhi: '午', na_yin: '路旁土' },
      yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
      ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
      shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
    }
    const wrapper = mount(BaziFullChartPanel, { props: { result } })

    expect(wrapper.findAll('.pillar')).toHaveLength(4)
    expect(wrapper.find('.san-yuan').exists()).toBe(false)
    expect(wrapper.find('.shensha').exists()).toBe(false)
    expect(wrapper.find('.rel-table').exists()).toBe(false)
    expect(wrapper.find('.analysis-grid').exists()).toBe(false)
    expect(wrapper.find('.dayun').exists()).toBe(false)
    expect(wrapper.find('.cs-grid').exists()).toBe(false)
    expect(wrapper.find('.ys').exists()).toBe(false)
  })

  it('applies focus and density to the stable public contract', async () => {
    const wrapper = mount(BaziFullChartPanel, {
      props: { result: sourceBackedBaziFull, focused: 'ri', density: 'compact' },
    })
    const pillars = wrapper.findAll('.pillar')

    expect(pillars[2].attributes('aria-current')).toBe('true')
    expect(pillars[2].attributes('aria-label')).toBe('日柱 壬申')
    expect(pillars.map(pillar => pillar.attributes('tabindex'))).toEqual(['-1', '-1', '-1', '-1'])
    expect(wrapper.classes()).toContain('engine-bazi-full-chart-panel--compact')

    await wrapper.setProps({ density: 'full', focused: 'shi' })
    expect(wrapper.classes()).toContain('engine-bazi-full-chart-panel--full')
    expect(pillars[3].attributes('data-focused')).toBe('true')
  })

  it('exposes the legacy annual composition point before yongshen', () => {
    const wrapper = mount(BaziFullChartPanel, {
      props: { result: sourceBackedBaziFull },
      slots: {
        'before-yongshen': '<div data-testid="composed-annual" />',
      },
    })
    const childClasses = [...(wrapper.element as HTMLElement).children]

    expect(wrapper.find('[data-testid="composed-annual"]').exists()).toBe(true)
    expect(childClasses.findIndex(child => child.querySelector('[data-testid="composed-annual"]')))
      .toBeLessThan(childClasses.findIndex(child => child.classList.contains('ys')))
  })
})
