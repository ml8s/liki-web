import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import type { BazhaiChart, BazhaiLayout } from '@shared/api/engine/fengshui'
import { LUOSHU_ORDER } from '@shared/utils/bazhai'

const result: BazhaiChart = {
  ming_gua: {
    gua: { name: '艮' },
    group: '西四命',
  },
  ba_zhai_dirs: {
    sheng_qi: '西南',
    tian_yi: '西北',
    yan_nian: '西',
    fu_wei: '东北',
    huo_hai: '南',
    wu_gui: '北',
    liu_sha: '东',
    jue_ming: '东南',
  },
}

function mountPanel(overrides: Partial<{ result: BazhaiChart; layout: BazhaiLayout | null; focused: string }> = {}) {
  return mount(BazhaiChartPanel, {
    props: {
      result,
      ...overrides,
    },
  })
}

describe('BazhaiChartPanel', () => {
  it('extracts MingGuaSection.vue and LiuNianFlySection.vue through FengshuiView.vue', () => {
    const wrapper = mountPanel()

    expect(wrapper.get('.mingua').text()).toBe('艮')
    expect(wrapper.get('.mg-name').text()).toBe('西四命')
    expect(wrapper.find('.mg-wuxing').exists()).toBe(false)

    const cells = wrapper.findAll('[role="gridcell"]')
    expect(cells).toHaveLength(8)
    expect(cells.filter(cell => cell.classes().includes('good'))).toHaveLength(4)
    expect(cells[0].text()).toContain('西南')
    expect(cells[0].attributes('data-focus-key')).toBe('sheng_qi')
    expect(wrapper.find('.sec-title').exists()).toBe(false)
    expect(wrapper.find('.fly-cell').exists()).toBe(false)
  })

  it('preserves the omitted annual-star branch and optional wuxing handling', () => {
    const annualResult: BazhaiChart = {
      ...result,
      liu_nian_xing: {
        year: 2026,
        ru_zhong: '七',
        gong_wei: LUOSHU_ORDER.map(gong_num => ({
          gong_num,
          xing_name: `星${gong_num}`,
        })),
      },
    }
    const wrapper = mountPanel({ result: annualResult })

    expect(wrapper.get('.sec-title').text()).toContain('2026')
    expect(wrapper.get('.ln-row').text()).toContain('七')

    const cells = wrapper.findAll('.chart-grid [role="gridcell"]')
    expect(cells.map(cell => cell.attributes('data-focus-key'))).toEqual(
      LUOSHU_ORDER.map(gongNum => `liu_nian_${gongNum}`),
    )
    expect(cells[0].text()).toContain('星4')
    expect(wrapper.find('.fly-wuxing').exists()).toBe(false)
  })

  it('preserves the Bazhai-layout nine-palace grid and badges in the chart family', () => {
    const layout: BazhaiLayout = {
      group: '西四命',
      ming_gua: '艮',
      door: { direction: '西南', gua_name: '坤', youxing: '生气', rating: '大吉', group: '西四卦', match: '吉' },
      master: { direction: '西', gua_name: '兑', youxing: '延年', rating: '吉', group: '西四卦', match: '吉' },
      stove: { direction: '西北', gua_name: '乾', youxing: '天医', rating: '吉', group: '西四卦', match: '吉' },
    }
    const wrapper = mountPanel({ layout })

    expect(wrapper.findAll('.bz-grid .bz-cell')).toHaveLength(9)
    expect(wrapper.find('.bz-grid .bz-cell.good .bz-youxing').text()).toBe('生气')
    expect(wrapper.findAll('.bz-badge').map(badge => badge.text())).toEqual(['门', '主卧', '灶'])
  })

  it('exposes focused domain cells without API, router, store, or session imports', () => {
    const wrapper = mountPanel({ focused: 'tian_yi' })
    const focused = wrapper.get('[data-focus-key="tian_yi"]')

    expect(focused.classes()).toContain('is-focused')
    expect(focused.attributes('aria-current')).toBe('true')
    expect(focused.attributes('tabindex')).toBe('0')
    expect(focused.attributes('aria-label')).toContain('西北')

    const source = readFileSync(
      resolve(process.cwd(), '../shared/components/engine/fengshui/BazhaiChartPanel.vue'),
      'utf8',
    )
    expect(source).toContain("defineProps<BazhaiChartPanelProps>()")
    expect(source).not.toMatch(/from ['"](?:@shared\/api|@\/api|@\/composables|vue-router|pinia)/)
    expect(source).not.toMatch(/\b(?:fetch|sessionStorage|localStorage)\b/)
  })

  it('keeps the natal and annual grids wrap-safe at 320px', () => {
    const source = readFileSync(
      resolve(process.cwd(), '../shared/components/engine/fengshui/BazhaiChartPanel.vue'),
      'utf8',
    )

    expect(source).toContain('min-width: 0')
    expect(source).toContain('repeat(4, minmax(0, 1fr))')
    expect(source).toContain('overflow-wrap: anywhere')
    expect(source).toContain('@media (max-width: 640px)')
  })
})
