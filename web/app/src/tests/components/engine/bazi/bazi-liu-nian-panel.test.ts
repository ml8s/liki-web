import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import type { BaziLiuNianOutput } from '@shared/api/engine/bazi'

// Provenance fixture: web/shared/components/charts/bazi/BaziLiunianDetail.vue,
// web/app/src/components/birth-profiles/ChartPreview.vue, and
// web/e2e/engine-fixtures.ts (baziLiunian).
const sourceBackedBaziLiunian: BaziLiuNianOutput = {
  year: 2026,
  nian_name: '丙午',
  shi_shen: '正官',
  nian_gan: '丙',
  nian_zhi: '午',
  wuxing: '火',
  na_yin: '天河水',
  sheng: 0,
  ke: 1,
  shensha: [{ name: '天乙贵人', category: '吉', description: '流运天乙贵人，有贵人相助' }],
  fuyin_fanyin: [],
  dayun_interactions: [],
  natal_interactions: [],
}

const interactionFixture: BaziLiuNianOutput = {
  ...sourceBackedBaziLiunian,
  fuyin_fanyin: [{ type: '伏吟', detail: '年柱伏吟', natal_index: 0 }],
  natal_interactions: [{
    gan_rels: [{ gan_a: '丙', relation: '丙克庚', type: '克' }],
    zhi_rels: [{ zhi_a: '午', zhi_b: '子', detail: '子午相冲', type: '冲' }, { zhi_a: '午', detail: '无特殊关系', type: '无' }],
  }],
  dayun_interactions: [{
    pillar_label: '壬午',
    gan_rels: [{ gan_a: '壬', relation: '壬克丙', type: '克' }],
  }],
}

const componentSource = readFileSync(
  resolve(process.cwd(), `../shared/components/engine/bazi/BaziLiuNianPanel.vue`),
  'utf8',
)
const sharedChartSource = readFileSync(
  resolve(process.cwd(), '../shared/styles/chart-panel.css'),
  'utf8',
)

describe('BaziLiuNianPanel | web/shared/components/charts/bazi/BaziLiunianDetail.vue + ChartPreview.vue', () => {
  it('renders annual shensha with semantic good/bad state and a named focus target', () => {
    const wrapper = mount(BaziLiuNianPanel, {
      props: { result: sourceBackedBaziLiunian, focused: 'liunian' },
    })
    const title = wrapper.find('.sec-title')

    expect(wrapper.find('.pillar.liunian').text()).toContain('流年 2026')
    expect(wrapper.find('.pillar.liunian').text()).toContain('丙午')
    expect(wrapper.find('.pillar.liunian').text()).toContain('正官')
    expect(wrapper.find('.pillar.liunian').text()).toContain('天河水')
    expect(wrapper.find('.pillar.liunian').text()).toContain('生数 0 · 克数 1')
    expect(wrapper.find('.liunian-detail').exists()).toBe(true)
    expect(wrapper.find('.chart-chip').classes()).toContain('good')
    expect(wrapper.find('.chart-chip').text()).toContain('天乙贵人')
    expect(wrapper.find('.chart-chip').attributes('title')).toBe('流运天乙贵人，有贵人相助')
    expect(title.attributes('id')).toBe(wrapper.attributes('aria-labelledby'))
    expect(wrapper.attributes('aria-current')).toBe('true')
    expect(wrapper.attributes('data-focused')).toBe('true')
    expect(wrapper.attributes('tabindex')).toBe('-1')
  })

  it('preserves fuyin and natal/dayun interaction groups in source order', () => {
    const wrapper = mount(BaziLiuNianPanel, { props: { result: interactionFixture } })
    const blocks = wrapper.findAll('.ln-block-title')
    const relRows = wrapper.findAll('.rel-row')

    expect(wrapper.findAll('.shensha')[1].text()).toContain('年柱伏吟')
    expect(blocks.map(block => block.text())).toEqual(['与命局', '与大运 · 壬午'])
    expect(relRows).toHaveLength(3)
    expect(relRows[0].text()).toContain('丙克庚')
    expect(relRows[1].text()).toContain('子午相冲')
    expect(relRows[1].text()).not.toContain('无特殊关系')
    expect(relRows[2].text()).toContain('壬克丙')
  })

  it('renders only the annual title when every optional projection is absent', () => {
    const result: BaziLiuNianOutput = {
      year: 2026,
      nian_name: '丙午',
      shi_shen: '正官',
    }
    const wrapper = mount(BaziLiuNianPanel, { props: { result } })

    expect(wrapper.find('.sec-title').exists()).toBe(true)
    expect(wrapper.find('.pillar.liunian').text()).toContain('丙午')
    expect(wrapper.find('.shensha').exists()).toBe(false)
    expect(wrapper.find('.rel-table').exists()).toBe(false)
    expect(wrapper.attributes('data-focused')).toBe('false')
    expect(wrapper.attributes('aria-current')).toBeUndefined()
  })

  it('exposes density and the 320px wrapping contract without horizontal fixed widths', () => {
    const wrapper = mount(BaziLiuNianPanel, {
      props: { result: sourceBackedBaziLiunian, density: 'compact' },
    })

    expect(wrapper.classes()).toContain('engine-bazi-liunian-panel--compact')
    expect(componentSource).toContain('.engine-bazi-liunian-panel { min-width: 0; }')
    expect(componentSource).toContain('flex-direction: column')
    expect(sharedChartSource).toContain('flex-wrap: wrap')
    expect(componentSource).not.toMatch(/width:\s*\d{3,}px/)
  })
})
