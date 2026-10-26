import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BazhaiLayoutPanel from '@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'
import type { BazhaiLayout } from '@shared/api/engine/fengshui'

const result: BazhaiLayout = {
  group: '西四命',
  ming_gua: '艮',
  door: {
    direction: '西南',
    gua_name: '坤',
    youxing: '生气',
    rating: '大吉',
    group: '西四卦',
    match: '吉',
  },
  master: {
    direction: '西',
    gua_name: '兑',
    youxing: '延年',
    rating: '吉',
    group: '西四卦',
    match: '凶',
  },
  stove: {
    direction: '西北',
    gua_name: '乾',
    youxing: '天医',
    rating: '平',
    group: '西四卦',
    match: '未知',
  },
}

describe('BazhaiLayoutPanel', () => {
  it('extracts BazhaiLayoutPanel.vue door, master, and stove verdicts in source order', () => {
    const wrapper = mount(BazhaiLayoutPanel, { props: { result } })

    expect(wrapper.get('.bz-gua-name').text()).toBe('艮')
    expect(wrapper.get('.bz-group').text()).toBe('西四命')

    const rows = wrapper.findAll('.bz-table tbody tr')
    expect(rows.map(row => row.get('[scope="row"]').text())).toEqual(['门', '主卧', '灶'])
    expect(rows[0].text()).toContain('西南')
    expect(rows[0].text()).toContain('生气')
    expect(rows[0].text()).toContain('大吉')
    expect(rows[0].findAll('.bz-good')[1].text()).toBe('吉')
    expect(rows[1].findAll('.bz-bad')[1].text()).toBe('凶')
    expect(rows[2].text()).toContain('未知')
    expect(wrapper.get('.bz-note').text()).toContain('门、主卧宜吉方')
  })


  it('exposes the focused verdict row without forbidden runtime boundaries', () => {
    const wrapper = mount(BazhaiLayoutPanel, { props: { result, focused: 'master' } })
    const focused = wrapper.get('[data-focus-key="master"]')

    expect(focused.classes()).toContain('is-focused')
    expect(focused.attributes('aria-current')).toBe('true')
    expect(focused.attributes('tabindex')).toBe('0')

    const source = readFileSync(
      resolve(process.cwd(), '../shared/components/engine/fengshui/BazhaiLayoutPanel.vue'),
      'utf8',
    )
    expect(source).toContain("defineProps<BazhaiLayoutPanelProps>()")
    expect(source).not.toMatch(/from ['"](?:@shared\/api|[^'"]*\/api\/|@\/composables|vue-router|pinia)/)
    expect(source).not.toMatch(/\b(?:fetch|sessionStorage|localStorage)\b/)
  })

  it('keeps the six-column verdict table usable at 320px', () => {
    const source = readFileSync(
      resolve(process.cwd(), '../shared/components/engine/fengshui/BazhaiLayoutPanel.vue'),
      'utf8',
    )

    expect(source).toContain('overflow-x: auto')
    expect(source).toContain('table-layout: fixed')
    expect(source).toContain('overflow-wrap: anywhere')
    expect(source).toContain('white-space: normal')
  })
})
