import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaziYongShenPanel from '@shared/components/engine/bazi/BaziYongShenPanel.vue'
import type { BaziYongShenResult } from '@shared/api/engine/bazi'

const fullResult: BaziYongShenResult = {
  selected: {
    yong: '木',
    xi: ['水', '木'],
    ji: ['土'],
    source: 'tiao_hou',
    confidence: 'high',
    known_fields: ['yong', 'xi', 'ji'],
  },
  status: 'resolved',
  conflicts: [],
  rule_id: 'YS-TH-1',
  policy_id: 'YS-POLICY-1',
  chart_digest: 'chart-1',
  yongshen_digest: 'yongshen-1',
  ge_ju_context: {
    ge_ju: '正财格',
    pattern_god_ten_god: '正财',
    pattern_god_source: 'yue',
    yong_fa: '顺用',
    clear: true,
    theme: 'support_control',
  },
  evidence: [
    {
      source: 'fu_yi',
      qiangruo: '身旺',
      xi: '水',
      ji: '土',
    },
    {
      source: 'tiao_hou',
      primary: '木',
      secondary: null,
    },
  ],
}

function mountPanel(props: {
  result: BaziYongShenResult
  focused?: string
  density?: 'compact' | 'full'
}) {
  return mount(BaziYongShenPanel, { props })
}

/**
 * Provenance: web/shared/components/charts/YongShenPanel.vue and
 * web/shared/components/charts/bazi/BaziYongShen.vue.
 */
describe('BaziYongShenPanel (YongShenPanel.vue + bazi/BaziYongShen.vue provenance)', () => {
  it('renders selected, favorable, unfavorable, and evidence projections in source order', () => {
    const wrapper = mountPanel({ result: fullResult })

    expect(wrapper.find('.engine-yongshen').attributes('data-status')).toBe('resolved')
    expect(wrapper.find('.ys-value').text()).toBe('木')
    expect(wrapper.find('.ys-source').text()).toBe('tiao_hou')
    expect(wrapper.findAll('.ys-chip.good').map(node => node.text())).toEqual(['水', '木'])
    expect(wrapper.findAll('.ys-chip.bad').map(node => node.text())).toEqual(['土'])
    expect(wrapper.findAll('.ys-evidence__row .ys-label').map(node => node.text()))
      .toEqual(['fu_yi', 'tiao_hou'])
    expect(wrapper.find('.ys-evidence__row').text()).toContain('身旺')
    expect(wrapper.find('.ys-evidence__row').text()).toContain('水')
    expect(wrapper.find('.ys-evidence__row').text()).toContain('土')
    expect(wrapper.find('.ys-evidence__row + .ys-evidence__row').text()).toContain('木')
    expect(wrapper.find('.ys-review').exists()).toBe(false)
  })

  it('keeps absent optional evidence and favor/unfavor collections explicit without crashing', () => {
    const absentResult = {
      ...fullResult,
      selected: {
        ...fullResult.selected,
        yong: null,
        xi: undefined,
        ji: undefined,
        source: '',
        confidence: 'high' as const,
      },
      evidence: undefined,
      conflicts: undefined,
      status: 'resolved' as const,
    } as unknown as BaziYongShenResult

    const wrapper = mountPanel({ result: absentResult })

    expect(wrapper.find('.ys-value').text()).toBe('—')
    expect(wrapper.find('.ys-source').exists()).toBe(false)
    expect(wrapper.find('.ys-chip').exists()).toBe(false)
    expect(wrapper.find('.ys-evidence').exists()).toBe(false)
    expect(wrapper.find('.ys-conflicts').exists()).toBe(false)
    expect(wrapper.find('.ys-review').exists()).toBe(true)
  })

  it('renders conflicts and review state, and marks the focused result domain', () => {
    const wrapper = mountPanel({
      result: {
        ...fullResult,
        status: 'needs_review',
        selected: {
          ...fullResult.selected,
          confidence: 'low',
        },
        conflicts: [{
          code: 'tiaohou_fuyi_conflict',
          message: '调候用神与扶抑忌神直接相反，不硬判',
        }],
      },
      focused: 'tiaohou_fuyi_conflict',
    })

    expect(wrapper.find('.engine-yongshen').attributes('data-status')).toBe('needs_review')
    expect(wrapper.find('.engine-yongshen__status').text()).toBe('needs_review')
    expect(wrapper.find('[role="note"]').text()).toContain('复核')
    expect(wrapper.find('[data-code="tiaohou_fuyi_conflict"]').classes())
      .toContain('is-focused')
    expect(wrapper.find('[data-code="tiaohou_fuyi_conflict"]').attributes('aria-current'))
      .toBe('true')
  })

  it('applies the compact density class', () => {
    const wrapper = mountPanel({ result: fullResult, density: 'compact' })

    expect(wrapper.find('.engine-yongshen--compact').exists()).toBe(true)
  })

  it('uses wrapping constraints for long values at the 320px floor', () => {
    const wrapper = mountPanel({
      result: {
        ...fullResult,
        selected: {
          ...fullResult.selected,
          xi: ['非常长的喜神值用于检查三百二十像素宽度下的自然换行约束'],
        },
        evidence: [{
          source: 'fu_yi',
          detail: '非常长的证据内容用于检查三百二十像素宽度下的自然换行约束和最小宽度行为',
        }],
      },
    })

    const rootStyle = wrapper.attributes('class')
    expect(wrapper.classes()).toContain('engine-yongshen')
    expect(rootStyle).toContain('engine-yongshen')
    expect(wrapper.find('.ys-chips').exists()).toBe(true)
    expect(wrapper.find('.ys-chip').text()).toContain('自然换行约束')
    expect(wrapper.find('.ys-evidence__body').text()).toContain('最小宽度行为')
  })
})
