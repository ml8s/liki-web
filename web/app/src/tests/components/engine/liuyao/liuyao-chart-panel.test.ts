import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import LiuyaoChartPanel from '@shared/components/engine/liuyao/LiuyaoChartPanel.vue'
import type { LiuyaoChartResult } from '@shared/api/engine/liuyao'

const divinationViewChart: LiuyaoChartResult = {
  name: '水火既济',
  ben_gua: '水火既济',
  bian_gua: '水火未济',
  gong: '坎',
  gong_wuxing: '水',
  lines: [
    { position: 1, liu_qin: '兄弟', zhi: '子', wuxing: '水', gan: '戊' },
    { position: 2, liu_qin: '官鬼', zhi: '丑', wuxing: '土', gan: '戊' },
    { position: 3, liu_qin: '妻财', zhi: '卯', wuxing: '木', gan: '戊' },
    { position: 4, liu_qin: '兄弟', zhi: '午', wuxing: '火', gan: '己' },
    { position: 5, liu_qin: '子孙', zhi: '未', wuxing: '土', gan: '己' },
    { position: 6, liu_qin: '父母', zhi: '申', wuxing: '金', gan: '己' },
  ],
  bian_yao: [
    { position: 1, liu_qin: '父母', zhi: '酉', gan: '己' },
    { position: 2, liu_qin: '兄弟', zhi: '戌', gan: '己' },
    { position: 3, liu_qin: '子孙', zhi: '亥', gan: '己' },
    { position: 4, liu_qin: '父母', zhi: '酉', gan: '戊' },
    { position: 5, liu_qin: '兄弟', zhi: '戌', gan: '戊' },
    { position: 6, liu_qin: '子孙', zhi: '亥', gan: '戊' },
  ],
  ri_chen_gan: '戊',
  ri_chen_zhi: '子',
  yue_jian_gan: '甲',
  yue_jian_zhi: '寅',
  yong_shen: { name: '妻财', position: 3, wang_shuai: '相' },
  gua_ci: { name: '水火既济', gua_ci: '亨，小利贞。', yao_ci: ['', '', '', '', '', ''] },
  wang_shuai: ['相', '休', '旺', '相', '休', '囚'],
  ri_chen_relations: [
    { relations: ['平'] },
    { relations: ['克'] },
    { relations: ['生'] },
    { relations: ['平'] },
    { relations: ['克'] },
    { relations: ['生'] },
  ],
  xun_kong: ['午', '未'],
  timing_candidates: [
    { id: 'timing_1', mechanism: '值期', window: '午月', trigger_branch: '午' },
  ],
  patterns: [{ type: '空', sub_type: '真空', position: 3, is_true: true }],
  casting: {
    mode: 'auto',
    order: 'bottom_up',
    yaos: [7, 8, 9, 7, 8, 7],
    dong_yao: [3],
    casting_id: 'cast_chart_test',
  },
}

describe('LiuyaoChartPanel — source: web/app/src/views/DivinationView.vue', () => {
  it('preserves the complete deterministic chart projection and block order', () => {
    const wrapper = mount(LiuyaoChartPanel, {
      props: { result: divinationViewChart },
    })

    expect(wrapper.get('h2').text()).toBe('卦象')
    expect(wrapper.get('.gua-name').text()).toBe('水火既济')
    expect(wrapper.get('.gua-sub').text()).toContain('水火未济')
    expect(wrapper.get('.gua-info').text()).toContain('坎宫')
    expect(wrapper.get('.gua-info').text()).toContain('日辰 戊子')
    expect(wrapper.get('.gua-info').text()).toContain('月建 甲寅')
    expect(wrapper.get('.gua-info').text()).toContain('旬空 午 未')
    expect(wrapper.findAll('.lines-sec')).toHaveLength(2)
    expect(wrapper.findAll('.lines-table')[0].findAll('.line-row')).toHaveLength(7)
    expect(wrapper.findAll('.line-row')[1].text()).toContain('6爻')
    expect(wrapper.findAll('.line-row')[4].classes()).toContain('dong')
    expect(wrapper.findAll('.lines-table')[1].findAll('.line-row')).toHaveLength(6)
    expect(wrapper.get('.rel-text').text()).toContain('妻财 · 第3爻')
    expect(wrapper.text()).toContain('卦辞')
    expect(wrapper.text()).toContain('真空')
    expect(wrapper.text()).toContain('应期')
    expect(wrapper.text()).toContain('午月：值期：午')
  })

  it('keeps absent optional chart values as explicit absent sections', () => {
    const emptyChart: LiuyaoChartResult = {
      ...divinationViewChart,
      bian_gua: undefined,
      yong_shen: {},
      gua_ci: { name: '', gua_ci: '', yao_ci: ['', '', '', '', '', ''] },
      timing_candidates: undefined,
      patterns: undefined,
    }
    const wrapper = mount(LiuyaoChartPanel, {
      props: { result: emptyChart, density: 'compact' },
    })

    expect(wrapper.find('.gua-sub').text()).not.toContain('变卦')
    expect(wrapper.findAll('.lines-sec')).toHaveLength(2)
    expect(wrapper.find('.rel-sec').exists()).toBe(false)
  })
})
