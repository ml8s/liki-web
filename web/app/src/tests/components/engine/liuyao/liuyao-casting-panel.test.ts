import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import LiuyaoCastingPanel from '@shared/components/engine/liuyao/LiuyaoCastingPanel.vue'
import type { LiuyaoCasting } from '@shared/api/engine/liuyao'

const divinationCastPanelCasting = {
  mode: 'coins',
  order: 'bottom_up',
  yaos: [9, 7, 8, 7, 8, 7],
  dong_yao: [1],
  casting_id: 'cast_test_1',
} satisfies LiuyaoCasting

describe('LiuyaoCastingPanel — source: web/shared/components/charts/DivinationCastPanel.vue', () => {
  it('preserves the baseline casting receipt projection, bottom-up order, and moving-line cue', () => {
    const wrapper = mount(LiuyaoCastingPanel, {
      props: {
        result: divinationCastPanelCasting,
        focused: 'line-6',
        density: 'full',
      },
    })

    expect(wrapper.get('.gua-name').text()).toBe('火泽睽')
    expect(wrapper.get('.gua-sub').text()).toContain('火水未济')
    expect(wrapper.findAll('.yao')).toHaveLength(6)
    expect(wrapper.get('.yao .yao-idx').text()).toBe('6')
    expect(wrapper.get('.yao .yao-shape').classes()).toContain('yang')
    expect(wrapper.get('.yao .yao-dong').text()).toBe('动爻')
  })

  it('omits the moving cue and changed hexagram when every line is static', () => {
    const wrapper = mount(LiuyaoCastingPanel, {
      props: {
        result: {
          ...divinationCastPanelCasting,
          yaos: [7, 7, 8, 7, 8, 7],
          dong_yao: [],
          casting_id: 'cast_static',
        },
      },
    })

    expect(wrapper.get('.gua-name').text()).toBe('火泽睽')
    expect(wrapper.find('.gua-sub').exists()).toBe(false)
    expect(wrapper.find('.yao-dong').exists()).toBe(false)
  })
})
