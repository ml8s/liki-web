import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import HuangliDaysPanel from '@shared/components/engine/huangli/HuangliDaysPanel.vue'
import type { HuangliDay } from '@shared/api/engine/huangli'

const huangliDaysPanelDays = [
  {
    date: '2026-06-06',
    ri_zhu: { gan: '辛', zhi: '酉', na_yin: '石榴木' },
    na_yin: '石榴木',
    wuxing: '金',
    jian_chu: '建',
    huangdao: { index: 1, name: '青龙', path: '黄道', sequence: 1 },
    xi_shen: '寅',
    cai_shen: '卯',
    fu_shen: '午',
    gan_ji: '甲不开仓',
    zhi_ji: '酉不宴客',
    mansion: { index: 1, name: '角', dong_wu: '蛟', wuxing: '木', group: '东方', group_idx: 1 },
    jie_qi: '芒种',
    jie_qi_days: 1,
    ren_yuan: '子',
    shi_chen: [{ zhi: '子', time: '23:00', huangdao: '青龙', jian_chu: '建', suitable: true }],
    suitability: 'recommended',
    reason: '宜嫁娶',
  },
  {
    date: '2026-06-07',
    ri_zhu: { gan: '壬', zhi: '戌', na_yin: '大海水' },
    na_yin: '大海水',
    wuxing: '水',
    jian_chu: '除',
    huangdao: { index: 2, name: '明堂', path: '黄道', sequence: 2 },
    xi_shen: '卯',
    cai_shen: '辰',
    fu_shen: '未',
    gan_ji: '',
    zhi_ji: '',
    mansion: { index: 2, name: '亢', dong_wu: '龙', wuxing: '金', group: '东方', group_idx: 1 },
    jie_qi: '',
    jie_qi_days: 0,
    ren_yuan: '',
    suitability: 'possible',
    reason: '次吉',
  },
  {
    date: '2026-06-08',
    ri_zhu: { gan: '癸', zhi: '亥', na_yin: '大海水' },
    na_yin: '大海水',
    wuxing: '水',
    jian_chu: '满',
    huangdao: { index: 3, name: '天刑', path: '黑道', sequence: 3 },
    xi_shen: '辰',
    cai_shen: '巳',
    fu_shen: '申',
    gan_ji: '',
    zhi_ji: '',
    mansion: { index: 3, name: '氐', dong_wu: '貉', wuxing: '土', group: '东方', group_idx: 1 },
    jie_qi: '',
    jie_qi_days: 0,
    ren_yuan: '',
    suitability: 'unsuitable',
    reason: '冲煞',
  },
] satisfies HuangliDay[]

describe('HuangliDaysPanel — source: web/shared/components/charts/HuangliDaysPanel.vue', () => {
  it('groups recommended, possible, and unsuitable days in source order', () => {
    const wrapper = mount(HuangliDaysPanel, {
      props: { result: huangliDaysPanelDays, focused: '2026-06-06', density: 'full' },
    })

    expect(wrapper.get('.range-info').text()).toContain('2026-06-06')
    expect(wrapper.get('.range-info').text()).toContain('2026-06-08')
    expect(wrapper.findAll('.sec-title').map(node => node.text())).toEqual(['推荐', '可选', '不宜'])
    expect(wrapper.findAll('.day.good')).toHaveLength(1)
    expect(wrapper.findAll('.day.possible')).toHaveLength(1)
    expect(wrapper.findAll('.day.bad')).toHaveLength(1)
    expect(wrapper.get('.day.good .d-summary').text()).toContain('2026-06-06')
    expect(wrapper.get('.day.good .d-summary').text()).toContain('辛酉')
    expect(wrapper.get('.day.good .d-summary').text()).toContain('建 · 宜嫁娶')
  })

  it('expands a day into its same-family detail', async () => {
    const wrapper = mount(HuangliDaysPanel, {
      props: { result: huangliDaysPanelDays },
    })
    const detail = wrapper.find<HTMLDetailsElement>('details.day.good')
    expect(detail.exists()).toBe(true)

    expect(detail.get('.d-detail').isVisible()).toBe(false)
    detail.element.open = true
    await nextTick()

    expect(detail.get('.d-detail').isVisible()).toBe(true)
    expect(detail.get('.d-detail').text()).toContain('时辰吉凶')
  })

  it('renders the source empty branch when no recommended day exists', () => {
    const wrapper = mount(HuangliDaysPanel, {
      props: { result: [] as HuangliDay[] },
    })

    expect(wrapper.get('.hint').text()).toBe('该范围内没有推荐吉日。')
    expect(wrapper.find('.day').exists()).toBe(false)
    expect(wrapper.find('.range-info').exists()).toBe(false)
  })
})
