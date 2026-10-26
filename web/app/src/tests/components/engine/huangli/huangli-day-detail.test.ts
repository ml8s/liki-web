import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import HuangliDayDetail from '@shared/components/engine/huangli/HuangliDayDetail.vue'
import type { HuangliDay } from '@shared/api/engine/huangli'

const huangliDayDetailSourceDay = {
  date: '2026-06-06',
  ri_zhu: { gan: '辛', zhi: '酉', na_yin: '石榴木' },
  na_yin: '石榴木',
  wuxing: '金',
  jian_chu: '建',
  huangdao: { index: 1, name: '青龙', path: '黄道', sequence: 1 },
  xi_shen: '正东',
  cai_shen: '东南',
  fu_shen: '午',
  gan_ji: '甲不开仓',
  zhi_ji: '酉不宴客',
  mansion: { index: 1, name: '角', dong_wu: '蛟', wuxing: '木', group: '东方', group_idx: 1 },
  jie_qi: '芒种',
  jie_qi_days: 1,
  ren_yuan: '子',
  shi_chen: [
    { zhi: '子', time: '23:00', huangdao: '青龙', jian_chu: '建', suitable: true },
    { zhi: '丑', time: '01:00', huangdao: '天刑', jian_chu: '除', suitable: false },
  ],
  suitability: 'recommended',
  reason: '宜嫁娶',
} satisfies HuangliDay

describe('HuangliDayDetail — source: web/shared/components/charts/HuangliDayDetail.vue', () => {
  it('projects the full day facts and localized shichen details', () => {
    const wrapper = mount(HuangliDayDetail, {
      props: { day: huangliDayDetailSourceDay, focused: '2026-06-06', density: 'full' },
    })

    expect(wrapper.findAll('.d-item')).toHaveLength(7)
    expect(wrapper.get('.d-grid').text()).toContain('黄道')
    expect(wrapper.findAll('.d-item')[1].text()).toContain('角 · 东方 · 木')
    expect(wrapper.get('.d-grid').text()).toContain('喜神 正东')
    expect(wrapper.get('.d-grid').text()).toContain('财神 东南')
    expect(wrapper.get('.d-grid').text()).toContain('芒种 · 第 1 天')
    expect(wrapper.get('.d-pengzu').text()).toContain('甲不开仓 · 酉不宴客')
    expect(wrapper.findAll('.sc-chip')).toHaveLength(2)
    expect(wrapper.get('.sc-chip.good').text()).toContain('吉')
    expect(wrapper.findAll('.sc-chip')[1].classes()).not.toContain('good')
    expect(wrapper.findAll('.sc-chip')[1].text()).toContain('凶')
  })

  it('keeps empty optional facts and absent shichen explicit', () => {
    const emptyDay: HuangliDay = {
      date: '2026-06-07',
      ri_zhu: { gan: '', zhi: '', na_yin: '' },
      na_yin: '',
      wuxing: '',
      jian_chu: '',
      huangdao: { index: 0, name: '', path: '', sequence: 0 },
      xi_shen: '',
      cai_shen: '',
      fu_shen: '',
      gan_ji: '',
      zhi_ji: '',
      mansion: { index: 0, name: '', dong_wu: '', wuxing: '', group: '', group_idx: 0 },
      jie_qi: '',
      jie_qi_days: 0,
      ren_yuan: '',
      suitability: 'unspecified',
    }
    const wrapper = mount(HuangliDayDetail, {
      props: { day: emptyDay },
    })

    // HuangliDay requires the day/huangdao object, so its empty projection is
    // preserved; every computed-only optional section remains absent.
    expect(wrapper.findAll('.d-grid > .d-item')).toHaveLength(1)
    expect(wrapper.find('.d-pengzu').exists()).toBe(false)
    expect(wrapper.find('.d-shichen').exists()).toBe(false)
  })
})
