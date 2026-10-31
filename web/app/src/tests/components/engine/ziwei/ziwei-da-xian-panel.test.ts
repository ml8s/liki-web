import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ZiweiDaXianItem } from '@shared/api/engine/ziwei'
import ZiweiDaXianPanel from '@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'

const provenance = [
  'web/shared/components/charts/ZiweiPanel.vue daxian section',
  'web/app/src/components/birth-profiles/ChartPreview.vue daXianRaw composition',
] as const

function daXian(currentYear = new Date().getFullYear()): ZiweiDaXianItem[] {
  return [
    {
      name: '甲限',
      gong: '命宫',
      qi_sui: 4,
      zhi_sui: 13,
      start_year: currentYear - 1,
      end_year: currentYear + 1,
    },
    { name: '乙限', gong: '父母', qi_sui: 14, zhi_sui: 23 },
  ]
}

describe(`ZiweiDaXianPanel extracted from ${provenance[0]} and ${provenance[1]}`, () => {
  it('projects names, ages, year ranges, and the current-year marker in order', () => {
    const wrapper = mount(ZiweiDaXianPanel, { props: { result: daXian() } })
    const periods = wrapper.findAll('.dy')

    expect(periods.map(period => period.get('.dy-name').text())).toEqual(['甲限', '乙限'])
    expect(periods.map(period => period.get('.dy-shen').text())).toEqual(['4–13岁', '14–23岁'])
    expect(periods.map(period => period.get('.dy-year').text())).toEqual([
      `${new Date().getFullYear() - 1}–${new Date().getFullYear() + 1}`,
      '',
    ])
    expect(periods[0].classes()).toContain('cur')
    expect(periods[0].attributes('aria-current')).toBe('date')
    expect(periods[1].classes()).not.toContain('cur')
    expect(periods[1].attributes('aria-current')).toBeUndefined()
  })

  it('keeps omitted year bounds explicit and empty', () => {
    const wrapper = mount(ZiweiDaXianPanel, {
      props: {
        result: [{ name: '丙限', gong: '福德', qi_sui: 24, zhi_sui: 33 }],
      },
    })

    expect(wrapper.get('.dy-year').text()).toBe('')
    expect(wrapper.get('.dy').classes()).not.toContain('cur')
  })

  it('projects the stable focused key and density class', () => {
    const wrapper = mount(ZiweiDaXianPanel, {
      props: { result: daXian(), focused: '乙限-14', density: 'compact' },
    })
    const periods = wrapper.findAll('.dy')

    expect(wrapper.classes()).toContain('engine-panel--compact')
    expect(periods[0].classes()).not.toContain('focused')
    expect(periods[1].classes()).toContain('focused')
    expect(periods[1].attributes('aria-current')).toBe('true')
  })

  it('exposes focusable periods with combined accessible names', async () => {
    const wrapper = mount(ZiweiDaXianPanel, { props: { result: daXian() }, attachTo: document.body })
    const current = wrapper.find<HTMLLIElement>('.dy')
    expect(current.exists()).toBe(true)

    expect(wrapper.findAll('.dy').map(period => period.attributes('tabindex'))).toEqual(['0', '0'])
    expect(current.attributes('aria-label')).toContain('甲限')
    expect(current.attributes('aria-label')).toContain('4–13岁')

    current.element.focus()
    expect(document.activeElement).toBe(current.element)

    wrapper.unmount()
  })
})
