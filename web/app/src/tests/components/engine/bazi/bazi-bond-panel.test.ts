import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaziBondPanel from '@shared/components/engine/bazi/BaziBondPanel.vue'
import type { BaziBondResult, BaziChartData } from '@shared/api/engine/bazi'

const chartA: BaziChartData = {
  nian: { gan: '庚', zhi: '辰', na_yin: '白蜡金' },
  yue: { gan: '乙', zhi: '酉', na_yin: '泉中水' },
  ri: { gan: '甲', zhi: '辰', na_yin: '覆灯火' },
  shi: { gan: '丙', zhi: '子', na_yin: '涧下水' },
  birth_year: 1990,
  gender: 'male',
  zi_shi_rule: '晚子时属本日',
}

const chartB: BaziChartData = {
  nian: { gan: '庚', zhi: '辰', na_yin: '白蜡金' },
  yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  ri: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  shi: { gan: '甲', zhi: '午', na_yin: '沙中金' },
  birth_year: 1992,
  gender: 'female',
  zi_shi_rule: '晚子时属本日',
}

const fullResult = {
  day_master: {
    a: { gan: '甲', gender: 'male', wuxing: '木' },
    b: { gan: '辛', gender: 'female', wuxing: '金' },
    gan_guan_xi: { gan_a: '甲', gan_b: '辛', type: '相克', relation: '辛克甲' },
    b_present_in_a: false,
  },
  spouse_palace: {},
  zhu_cross: {
    pairs: [
      {
        a_zhu: 'ri',
        b_zhu: 'ri',
        a_gan: '甲',
        b_gan: '辛',
        a_zhi: '辰',
        b_zhi: '巳',
        gan_guan_xi: { gan_a: '甲', gan_b: '辛', type: '相克', relation: '辛克甲' },
        zhi_guan_xi: { zhi_a: '辰', zhi_b: '巳', type: '无', detail: '无特殊关系' },
      },
      {
        a_zhu: 'nian',
        b_zhu: 'nian',
        a_gan: '庚',
        b_gan: '庚',
        a_zhi: '辰',
        b_zhi: '辰',
        gan_guan_xi: { gan_a: '庚', gan_b: '庚', type: '相同', relation: '庚庚同气' },
        zhi_guan_xi: { zhi_a: '辰', zhi_b: '辰', type: '相刑', detail: '辰辰自刑' },
      },
      {
        a_zhu: 'yue',
        b_zhu: 'nian',
        a_gan: '乙',
        b_gan: '庚',
        a_zhi: '酉',
        b_zhi: '辰',
        gan_guan_xi: { gan_a: '乙', gan_b: '庚', type: '天干五合', relation: '乙庚合化金' },
        zhi_guan_xi: { zhi_a: '酉', zhi_b: '辰', type: '六合', detail: '辰酉合化金' },
      },
    ],
  },
  shi_shen_cross: {
    a_to_b: { nian_gan: '七杀', yue_gan: '劫财', ri_gan: '正官', shi_gan: '偏印' },
    b_to_a: { nian_gan: '劫财', yue_gan: '伤官', ri_gan: '正财', shi_gan: '比肩' },
  },
  wuxing_cross: {},
  nayin_cross: {
    pairs: [
      {
        a_zhu: 'ri',
        b_zhu: 'ri',
        a_na_yin: '覆灯火',
        b_na_yin: '白蜡金',
        relation: '火克金',
      },
    ],
  },
  spouse_star: {},
  shensha_cross: {
    tian_yi: { a_present_in_b: true, b_present_in_a: true },
    custom_star: { a_present_in_b: true, b_present_in_a: false },
  },
} as unknown as BaziBondResult

function mountPanel(props: {
  result: BaziBondResult
  chartA: BaziChartData
  chartB: BaziChartData
  focused?: string
  density?: 'compact' | 'full'
  nameA?: string
  nameB?: string
}) {
  return mount(BaziBondPanel, { props })
}

/**
 * Provenance: web/shared/components/charts/HepanPanel.vue and
 * web/app/src/views/HepanView.vue.
 */
describe('BaziBondPanel (HepanPanel.vue + HepanView.vue provenance)', () => {
  it('preserves two-chart, day-master, branch, nayin, shensha, and ten-god order', () => {
    const wrapper = mountPanel({
      result: fullResult,
      chartA,
      chartB,
      nameA: '张三',
      nameB: '李四',
    })

    expect(wrapper.findAll('.person').map(node => node.text())).toEqual(['张三', '李四'])
    expect(wrapper.find('[data-cell-id="A:nian"]').text()).toContain('庚辰')
    expect(wrapper.find('[data-cell-id="A:ri"]').text()).toContain('甲辰')
    expect(wrapper.find('[data-cell-id="B:shi"]').text()).toContain('甲午')

    expect(wrapper.find('.rel-text').text()).toBe('张三 日主 甲辰 ↔ 对方日主 辛巳 · 天干 辛克甲 · 地支 无特殊关系')
    const branchSection = wrapper.findAll('.rel-sec')
      .find(node => node.find('.sec-title').text() === '关键地支关系')
    expect(branchSection).toBeTruthy()
    expect(branchSection!.findAll('.rel-chip').map(node => node.text()))
      .toEqual(['年柱·庚辰 — 年柱·庚辰：辰辰自刑', '月柱·乙酉 — 年柱·庚辰：辰酉合化金'])
    expect(wrapper.findAll('.rel-chip.nayin').map(node => node.text()))
      .toEqual(['日柱·覆灯火 ↔ 日柱·白蜡金：火克金'])
    expect(wrapper.findAll('.rel-chip.shensha').map(node => node.text())).toEqual([
      '张三的天乙贵人入对方命盘',
      '对方的天乙贵人入张三命盘',
      '张三的custom_star入对方命盘',
    ])
    expect(wrapper.findAll('.shi-shen .rel-chip').map(node => node.text())).toEqual([
      '张三年柱看对方：七杀',
      '对方年柱看张三：劫财',
      '张三月柱看对方：劫财',
      '对方月柱看张三：伤官',
      '张三日柱看对方：正官',
      '对方日柱看张三：正财',
      '张三时柱看对方：偏印',
      '对方时柱看张三：比肩',
    ])
  })

  it('uses explicit display-name fallbacks without inventing either person', () => {
    const wrapper = mountPanel({
      result: fullResult,
      chartA,
      chartB,
      nameA: '  ',
    })

    expect(wrapper.findAll('.person').map(node => node.text())).toEqual(['档案 A', '对方'])
    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('renders omitted cross collections and absent optional pillar facts without crashing', () => {
    const minimalResult = {
      zhu_cross: { pairs: [] },
      nayin_cross: { pairs: [] },
      shi_shen_cross: {},
      shensha_cross: {},
    } as unknown as BaziBondResult
    const minimalChart = {
      nian: { gan: '', zhi: '', na_yin: '' },
      yue: {},
      ri: {},
      shi: {},
    } as unknown as BaziChartData

    const wrapper = mountPanel({
      result: minimalResult,
      chartA: minimalChart,
      chartB: minimalChart,
    })

    expect(wrapper.find('.rel-sec').exists()).toBe(false)
    expect(wrapper.findAll('.pillar .pl').map(node => node.text())).toEqual([
      '年柱',
      '月柱',
      '日柱',
      '时柱',
      '年柱',
      '月柱',
      '日柱',
      '时柱',
    ])
    expect(wrapper.find('[data-cell-id="B:yue"]').text()).not.toContain('undefined')
  })

  it('names every focusable pillar cell and marks the focused cell', () => {
    const wrapper = mountPanel({
      result: fullResult,
      chartA,
      chartB,
      nameA: '张三',
      nameB: '李四',
      focused: 'B:ri',
    })

    const cells = wrapper.findAll('.pillar[tabindex="0"]')
    expect(cells).toHaveLength(8)
    for (const cell of cells) {
      expect(cell.attributes('role')).toBe('group')
      expect(cell.attributes('aria-label')).toBeTruthy()
    }
    expect(cells[0]!.attributes('aria-label')).toBe('年柱 庚辰 白蜡金')
    expect(cells[6]!.attributes('aria-label')).toBe('日柱 辛巳 白蜡金')
    expect(cells[6]!.classes()).toContain('is-focused')
  })

  it('applies the compact density class', () => {
    const wrapper = mountPanel({
      result: fullResult,
      chartA,
      chartB,
      density: 'compact',
    })

    expect(wrapper.find('.engine-bond--compact').exists()).toBe(true)
  })

  it('uses wrapping constraints for long names and relations at the 320px floor', () => {
    const wrapper = mountPanel({
      result: fullResult,
      chartA,
      chartB,
      nameA: '非常长的档案名称用于检查三百二十像素宽度下的自然换行约束',
      nameB: '非常长的对方名称用于检查三百二十像素宽度下的自然换行约束',
    })

    const rootStyle = wrapper.attributes('class')
    expect(wrapper.classes()).toContain('engine-bond')
    expect(rootStyle).toContain('engine-bond')
    expect(wrapper.find('.rels').exists()).toBe(true)
    expect(wrapper.find('.person').text()).toContain('自然换行约束')
    expect(wrapper.text()).toContain('非常长的对方名称用于检查三百二十像素宽度下的自然换行约束')
  })
})
