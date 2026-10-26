import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ZiweiLiuNianData } from '@shared/api/engine/ziwei'
import ZiweiLiuNianPanel from '@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const componentSource = readFileSync(resolve(process.cwd(), '../shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'), 'utf8')


const provenance = [
  'web/shared/components/charts/ZiweiPanel.vue grid vocabulary',
  'web/app/src/components/birth-profiles/ChartPreview.vue liuNianPlaced composition',
] as const

function liuNian(): ZiweiLiuNianData {
  return {
    ming_gong: '命宫',
    ming_gong_name: '兄弟',
    zhi: '子',
    si_hua: { 紫微: '禄', 天府: '权' },
    si_hua_gong: { 紫微: '命宫' },
    gong_wei: [
      { zhi: '子', name: '兄弟', xing_yao: ['紫微', '天府'], is_liu_ming: true },
      { zhi: '丑', name: '夫妻' },
    ],
  }
}

describe(`ZiweiLiuNianPanel extracted from ${provenance[0]} and ${provenance[1]}`, () => {
  it('projects annual sihua, palace order, stars, and annual minggong', () => {
    const wrapper = mount(ZiweiLiuNianPanel, { props: { result: liuNian() } })
    const chips = wrapper.findAll('.zw-sihua-chip')
    const palaces = wrapper.findAll('.zw-palace')

    expect(wrapper.get('.sec-title').text()).toBe('流年')
    expect(chips.map(chip => chip.text())).toEqual(['紫微禄 命宫', '天府权'])
    expect(palaces.map(palace => palace.get('.zw-pname').text())).toEqual(['兄弟', '夫妻'])
    expect(palaces[0].get('.zw-main').text()).toBe('紫微 天府')
    expect(palaces[0].classes()).toContain('ming')
    expect(palaces[1].find('.zw-main').exists()).toBe(false)
    expect(wrapper.get('.zwc-row').text()).toContain('兄弟')
  })

  it('supports a minimal result with omitted optional annual fields', () => {
    const wrapper = mount(ZiweiLiuNianPanel, {
      props: {
        result: { ming_gong: '命宫', zhi: '子' },
        density: 'full',
      },
    })

    expect(wrapper.findAll('.zw-sihua-chip')).toHaveLength(0)
    expect(wrapper.findAll('.zw-palace')).toHaveLength(0)
    expect(wrapper.get('.zwc-row').text()).toContain('命宫')
    expect(wrapper.classes()).toContain('engine-panel--full')
  })

  it('projects both natural and explicit focus without losing the natural state', () => {
    const wrapper = mount(ZiweiLiuNianPanel, {
      props: { result: liuNian(), focused: '夫妻-丑', density: 'compact' },
    })
    const palaces = wrapper.findAll('.zw-palace')

    expect(wrapper.classes()).toContain('engine-panel--compact')
    expect(palaces[0].classes()).toContain('ming')
    expect(palaces[0].classes()).not.toContain('focused')
    expect(palaces[0].attributes('aria-current')).toBeUndefined()
    expect(palaces[1].classes()).toContain('focused')
    expect(palaces[1].attributes('aria-current')).toBe('true')
  })

  it('exposes focusable annual palaces with name and branch names', async () => {
    const wrapper = mount(ZiweiLiuNianPanel, { props: { result: liuNian() }, attachTo: document.body })
    const palace = wrapper.find<HTMLDivElement>('.zw-palace')
    expect(palace.exists()).toBe(true)

    expect(wrapper.findAll('.zw-palace').map(cell => cell.attributes('tabindex'))).toEqual(['0', '0'])
    expect(palace.attributes('aria-label')).toBe('兄弟 子')

    palace.element.focus()
    expect(document.activeElement).toBe(palace.element)

    wrapper.unmount()
  })

  it('keeps long annual palace text within the 320px floor', () => {
    const wrapper = mount(ZiweiLiuNianPanel, {
      props: {
        result: {
          ming_gong: '命宫',
          zhi: '子',
          gong_wei: [{
            zhi: '亥',
            name: `exceptionally-long-annual-palace-${'y'.repeat(40)}`,
            xing_yao: ['very-long-annual-star-name'],
          }],
        },
      },
    })

    expect(componentSource).toContain('@media (max-width: 320px)')
    expect(componentSource).toContain('overflow-wrap: anywhere')
    expect(wrapper.element.scrollWidth).toBeLessThanOrEqual(wrapper.element.clientWidth)
  })
})
