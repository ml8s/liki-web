import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import type { BaziChartData } from '@shared/api/engine/bazi'

// Provenance fixture: web/shared/components/charts/bazi/BaziPillars.vue and
// web/e2e/engine-fixtures.ts (baziChart), reduced to bazi_chart contract fields.
const sourceBackedBaziChart: BaziChartData = {
  nian: { gan: '庚', zhi: '午', na_yin: '路旁土' },
  yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1990,
  gender: 'female',
  zi_shi_rule: '晚子时属本日',
}

const componentSource = readFileSync(
  resolve(process.cwd(), `../shared/components/engine/bazi/BaziChartPanel.vue`),
  'utf8',
)

describe('BaziChartPanel | web/shared/components/charts/bazi/BaziPillars.vue, web/shared/components/charts/BaziPanel.vue', () => {
  it('renders the canonical four-pillar projection with accessible names', () => {
    const wrapper = mount(BaziChartPanel, { props: { result: sourceBackedBaziChart } })
    const pillars = wrapper.findAll('.pillar')

    expect(pillars).toHaveLength(4)
    expect(pillars.map(pillar => pillar.find('.gl').text())).toEqual(['庚午', '辛巳', '壬申', '癸卯'])
    expect(pillars.map(pillar => pillar.find('.nayin').text())).toEqual(['路旁土', '白蜡金', '剑锋金', '金箔金'])
    expect(pillars[2]).toBeTruthy()
    expect(pillars[2].attributes('aria-label')).toBe('日柱 壬申')
    expect(pillars.map(pillar => pillar.attributes('tabindex'))).toEqual(['-1', '-1', '-1', '-1'])
  })

  it('marks only the requested focused pillar without changing ordering', async () => {
    const wrapper = mount(BaziChartPanel, {
      props: { result: sourceBackedBaziChart, focused: 'ri' },
    })
    const pillars = wrapper.findAll('.pillar')

    expect(pillars.map(pillar => pillar.attributes('data-focused'))).toEqual(['false', 'false', 'true', 'false'])
    expect(pillars[2].classes()).toContain('focused')
    expect(pillars[2].attributes('aria-current')).toBe('true')
    expect(pillars[0].attributes('aria-current')).toBeUndefined()

    await wrapper.setProps({ focused: 'nian' })
    expect(pillars[0].attributes('data-focused')).toBe('true')
    expect(pillars[2].attributes('data-focused')).toBe('false')
  })

  it('keeps empty optional values explicit and omits the absent dayun branch', () => {
    const result: BaziChartData = {
      ...sourceBackedBaziChart,
      nian: { gan: '庚', zhi: '午', na_yin: '' },
    }
    const wrapper = mount(BaziChartPanel, { props: { result } })

    expect(wrapper.find('.pillars').exists()).toBe(true)
    expect(wrapper.find('.dayun').exists()).toBe(false)
    expect(wrapper.find('.pillar .nayin').text()).toBe('')
  })

  it('uses the density class and preserves the 320px two-column collapse contract', () => {
    const wrapper = mount(BaziChartPanel, {
      props: { result: sourceBackedBaziChart, density: 'compact' },
    })

    expect(wrapper.classes()).toContain('engine-bazi-chart-panel--compact')
    expect(componentSource).toContain('.engine-bazi-chart-panel { min-width: 0; }')
    expect(componentSource).toContain('@media (max-width: 640px)')
    expect(componentSource).toContain('grid-template-columns: repeat(2, 1fr)')
    expect(wrapper.find('.pillars').classes()).toContain('pillars')
  })
})
