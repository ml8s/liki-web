import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ZiweiChart } from '@shared/api/engine/ziwei'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const componentSource = readFileSync(resolve(process.cwd(), '../shared/components/engine/ziwei/ZiweiChartPanel.vue'), 'utf8')


const ziweiPanelSource = 'web/shared/components/charts/ZiweiPanel.vue'

function chart(overrides: Partial<ZiweiChart> = {}): ZiweiChart {
  return {
    gong_wei: [
      {
        index: '命宫',
        name: '命宫',
        gan: '甲',
        zhi: '子',
        is_shen_gong: true,
        xing_yao: [
          { xing: 'ziwei', name: '紫微', is_zhu_xing: true, si_hua: '禄', liang_du: '庙' },
          { xing: 'wenchang', name: '文昌', si_hua: '科' },
        ],
        za_yao: ['天魁'],
        chang_sheng: '长生',
        ages: [2, 11],
      },
      { index: '兄弟', name: '兄弟', zhi: '丑', xing_yao: [] },
    ],
    ming_gong: '命宫',
    ju_shu: '水二局',
    si_hua: { 紫微: '禄' },
    ...overrides,
  }
}

describe(`ZiweiChartPanel extracted from ${ziweiPanelSource}`, () => {
  it('renders the configured palace order, star classes, and center facts', () => {
    const wrapper = mount(ZiweiChartPanel, { props: { result: chart() } })
    const palaces = wrapper.findAll('.zw-palace')

    expect(palaces[0].text()).toContain('命宫甲子')
    expect(palaces[0].classes()).toContain('ming')
    expect(palaces[0].classes()).toContain('shen')
    expect(palaces[0].get('.zw-main').text()).toContain('紫微庙禄')
    expect(palaces[0].get('.zw-minor').text()).toContain('文昌科')
    expect(palaces[0].get('.zw-za').text()).toBe('天魁')
    expect(palaces[0].attributes('title')).toBe('小限 2、11')
    expect(palaces[1].find('.zw-main').exists()).toBe(false)
    expect(wrapper.get('.zwc-title').text()).toBe('水二局')
    expect(wrapper.get('.zwc-row').text()).toContain('命主')
    expect(wrapper.get('.zw-sihua-chip').text()).toContain('紫微禄')
  })

  it('keeps omitted optional palace and chart fields in explicit empty branches', () => {
    const wrapper = mount(ZiweiChartPanel, {
      props: {
        result: chart({
          ming_gong: undefined,
          ju_shu: undefined,
          si_hua: undefined,
          gong_wei: [{ index: '父母', name: '父母', zhi: '寅', xing_yao: [] }],
        }),
      },
    })

    expect(wrapper.findAll('.zw-main')).toHaveLength(0)
    expect(wrapper.findAll('.zw-minor')).toHaveLength(0)
    expect(wrapper.findAll('.zw-za')).toHaveLength(0)
    expect(wrapper.findAll('.zw-sihua-chip')).toHaveLength(0)
    expect(wrapper.get('.zwc-title').text()).toBe('')
  })

  it('projects focused and natural palace states without changing DTO order', () => {
    const wrapper = mount(ZiweiChartPanel, {
      props: { result: chart(), focused: '命宫', density: 'compact' },
    })
    const palaces = wrapper.findAll('.zw-palace')

    expect(wrapper.classes()).toContain('engine-panel--compact')
    expect(palaces.map(palace => palace.attributes('aria-label'))).toEqual([
      '命宫 甲子',
      '兄弟 丑',
    ])
    expect(palaces[0].classes()).toContain('focused')
    expect(palaces[0].attributes('aria-current')).toBe('true')
    expect(palaces[1].classes()).not.toContain('focused')
    expect(palaces[1].attributes('aria-current')).toBeUndefined()
  })

  it('exposes focusable palaces with real names and keyboard focus', async () => {
    const wrapper = mount(ZiweiChartPanel, { props: { result: chart() }, attachTo: document.body })
    const palace = wrapper.find<HTMLDivElement>('.zw-palace')
    expect(palace.exists()).toBe(true)

    expect(wrapper.findAll('.zw-palace').map(cell => cell.attributes('tabindex')))
      .toEqual(['0', '0'])
    expect(palace.attributes('aria-label')).toBe('命宫 甲子')

    palace.element.focus()
    expect(document.activeElement).toBe(palace.element)

    wrapper.unmount()
  })

  it('keeps long star and center text safe at the 320px floor', () => {
    const wrapper = mount(ZiweiChartPanel, {
      props: {
        result: chart({
          gong_wei: [{
            index: 'long',
            name: ' exceptionally-long-palace-name ',
            zhi: '亥',
            xing_yao: [{ xing: 'long-star', name: 'very-long-star-name-'.repeat(3) }],
          }],
          ju_shu: 'another-exceptionally-long-center-value',
        }),
      },
    })

    expect(wrapper.attributes('class')).toContain('engine-ziwei-panel')
    expect(componentSource).toContain('@media (max-width: 320px)')
    expect(componentSource).toContain('overflow-wrap: anywhere')
    expect(wrapper.element.scrollWidth).toBeLessThanOrEqual(wrapper.element.clientWidth)
  })
})
