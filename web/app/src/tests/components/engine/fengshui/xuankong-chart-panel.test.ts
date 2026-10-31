import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import XuankongChartPanel from '@shared/components/engine/fengshui/XuankongChartPanel.vue'
import type { XuankongChart } from '@shared/api/engine/fengshui'
import { LUOSHU_ORDER } from '@shared/utils/bazhai'

function star(number: number) {
  return { number, name: `星${number}` }
}

const result: XuankongChart = {
  yun: {
    year: 2026,
    yun_number: 9,
    yun_name: '九运',
    start_year: 2024,
    end_year: 2043,
  },
  gong_wei: LUOSHU_ORDER.map(gong_num => ({
    gong_num,
    yun_xing: star(9),
    shan_xing: star(gong_num),
    xiang_xing: star(9 - (gong_num % 3)),
  })),
  four_situation: {
    name: '双星会坐',
    sit_mountain_timely: true,
    face_facing_timely: false,
    sit_facing_timely: true,
    face_mountain_timely: false,
  },
  zuo_shan: 0,
  zuo_shan_name: '子',
  xiang_shan: 12,
  xiang_shan_name: '午',
  shou_shan_chu_sha: {
    zheng_shen: 9,
    ling_shen: 1,
    shou_shan: true,
    chu_sha: false,
  },
}

describe('XuankongChartPanel', () => {
  it('extracts XuankongPanel.vue period, palace order, facing/sitting, and four situations', () => {
    const wrapper = mount(XuankongChartPanel, { props: { result } })

    expect(wrapper.get('.xk-yuninfo').text()).toBe('九运（2024–2043）')
    expect(wrapper.get('.xk-sitface').text()).toContain('子')
    expect(wrapper.get('.xk-sitface').text()).toContain('午')

    const cells = wrapper.findAll('.chart-grid [role="gridcell"]')
    expect(cells.map(cell => cell.attributes('data-focus-key'))).toEqual(
      LUOSHU_ORDER.map(gongNum => `gong_${gongNum}`),
    )
    expect(cells[7].classes()).toContain('sit')
    expect(cells[1].classes()).toContain('face')
    expect(wrapper.get('.xk-situation').text()).toBe('双星会坐')
    expect(wrapper.findAll('.xk-timely-item')).toHaveLength(4)
    expect(wrapper.find('.xk-assessment').exists()).toBe(false)
    expect(wrapper.find('.chart-chip:not(.xk-timely-item)').exists()).toBe(false)
    expect(wrapper.find('.xk-combos').exists()).toBe(false)
  })

  it('renders absent optional assessment, layers, and combinations without crashing', () => {
    const partial: XuankongChart = {
      ...result,
      yun: { ...result.yun, yuan: undefined },
      gong_wei: [],
      fu_yin_layers: undefined,
      fan_yin_layers: undefined,
      xing_jia_hui: undefined,
      shou_shan_chu_sha: {
        ...result.shou_shan_chu_sha,
        assessment: undefined,
      },
    }
    const wrapper = mount(XuankongChartPanel, { props: { result: partial } })

    expect(wrapper.get('.xk-yuninfo').text()).toBe('九运（2024–2043）')
    expect(wrapper.findAll('[role="gridcell"]')).toHaveLength(0)
    expect(wrapper.get('.xk-situation').text()).toBe('双星会坐')
    expect(wrapper.findAll('.xk-timely-item')).toHaveLength(4)
  })

  it('preserves named combinations and focused palace accessibility', () => {
    const full: XuankongChart = {
      ...result,
      fu_yin_layers: [],
      fan_yin_layers: ['山星反吟'],
      xing_jia_hui: [
        { shan_num: 7, xiang_num: 2, name: '七二同道', meaning: '组合含义' },
        { shan_num: 2, xiang_num: 7, name: '星曜加会', meaning: 'fallback' },
      ],
      shou_shan_chu_sha: {
        ...result.shou_shan_chu_sha,
        assessment: '收山得宜',
      },
    }
    const wrapper = mount(XuankongChartPanel, { props: { result: full, focused: 'gong_4' } })
    const focused = wrapper.get('[data-focus-key="gong_4"]')

    expect(focused.classes()).toContain('is-focused')
    expect(focused.attributes('aria-current')).toBe('true')
    expect(focused.attributes('tabindex')).toBe('0')
    expect(focused.attributes('aria-label')).toContain('山星 4 星4')
    expect(wrapper.findAll('.xk-combo')).toHaveLength(1)
    expect(wrapper.get('.xk-assessment').text()).toBe('收山得宜')
    expect(wrapper.get('.xk-sub .chart-chip').text()).toContain('山星反吟')
  })
})
