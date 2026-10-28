import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ZiweiChart } from '@shared/api/engine/ziwei'
import ZiweiFullChartPanel from '@shared/components/engine/ziwei/ZiweiFullChartPanel.vue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const componentSource = readFileSync(resolve(process.cwd(), '../shared/components/engine/ziwei/ZiweiFullChartPanel.vue'), 'utf8')


const provenance = {
  grid: 'web/shared/components/charts/ZiweiPanel.vue',
  fullchart: 'web/shared/components/charts/ZiweiPanel.vue pattern/sanfang/kongjie sections',
} as const

function fullChart(): ZiweiChart {
  return {
    gong_wei: [{ index: '命宫', name: '命宫', zhi: '子', xing_yao: [] }],
    ming_gong: '命宫',
    patterns: [
      { name: '君臣庆会', description: 'source pattern description' },
      { name: '火贪格', description: 'second source pattern' },
    ],
    san_fang: [{
      name: '命宫三方',
      zhu_xing: ['紫微', '天府'],
      fu_xing: ['文昌'],
      si_hua: ['禄', '权'],
    }],
    kong_gong: [{
      gong_name: '夫妻',
      dui_gong: '官禄',
      jie_xing: ['天机', '巨门'],
    }],
  }
}

describe(`ZiweiFullChartPanel extracted from ${provenance.fullchart}`, () => {
  it('renders the basic grid before full-chart sections in source order', () => {
    const wrapper = mount(ZiweiFullChartPanel, { props: { result: fullChart() } })

    expect(wrapper.find('.zw-grid').exists()).toBe(true)
    expect(wrapper.get('.zw-palace').text()).toContain('命宫')
    expect(wrapper.findAll('.zw-section').map(section => section.get('.sec-title').text()))
      .toEqual(['格局', '三方四正', '空宫借星'])
    expect(wrapper.findAll('.chart-chip').map(chip => chip.text())).toEqual([
      '君臣庆会',
      '火贪格',
      '夫妻借官禄宫：天机 巨门',
    ])
    expect(wrapper.get('.zw-sf-body').text()).toContain('紫微 天府')
    expect(wrapper.get('.zw-sf-body').text()).toContain('文昌')
    expect(wrapper.get('.zw-sf-body').text()).toContain('禄')
  })

  it('renders the minimal ZiweiChart without optional full-chart branches', () => {
    const wrapper = mount(ZiweiFullChartPanel, {
      props: {
        result: {
          gong_wei: [{ index: '父母', name: '父母', zhi: '丑', xing_yao: [] }],
        },
        density: 'full',
      },
    })

    expect(wrapper.get('.zw-palace').text()).toContain('父母')
    expect(wrapper.findAll('.zw-section')).toHaveLength(0)
    expect(wrapper.classes()).toContain('engine-panel--full')
  })

  it('keeps focused projection on the composed basic grid', () => {
    const wrapper = mount(ZiweiFullChartPanel, {
      props: { result: fullChart(), focused: '命宫', density: 'compact' },
    })

    expect(wrapper.classes()).toContain('engine-panel--compact')
    expect(wrapper.get('.zw-palace').classes()).toContain('focused')
    expect(wrapper.get('.zw-palace').attributes('aria-current')).toBe('true')
  })

  it('exposes focusable palace cells from the source-backed grid', async () => {
    const wrapper = mount(ZiweiFullChartPanel, { props: { result: fullChart() }, attachTo: document.body })
    const palace = wrapper.find<HTMLDivElement>('.zw-palace')
    expect(palace.exists()).toBe(true)

    expect(palace.attributes('tabindex')).toBe('0')
    expect(palace.attributes('aria-label')).toBe('命宫 子')

    palace.element.focus()
    expect(document.activeElement).toBe(palace.element)

    wrapper.unmount()
  })

  it('does not synthesize a daxian section that the frozen DTO cannot carry', () => {
    const wrapper = mount(ZiweiFullChartPanel, { props: { result: fullChart() } })

    expect(wrapper.find('.zw-daxian').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('大限')
  })

  it(`preserves ${provenance.grid} hierarchy while wrapping at 320px`, () => {
    const wrapper = mount(ZiweiFullChartPanel, {
      props: {
        result: {
          ...fullChart(),
          patterns: [{
            name: 'exceptionally-long-pattern-name-for-the-320px-floor',
            description: 'description',
          }],
        },
      },
    })

    expect(componentSource).toContain('overflow-wrap: anywhere')
    expect(wrapper.element.scrollWidth).toBeLessThanOrEqual(wrapper.element.clientWidth)
  })
})
