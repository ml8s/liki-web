import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DivinationView from '@/views/DivinationView.vue'
import type { LiuyaoCasting, LiuyaoChartResult } from '@shared/api/engine/liuyao'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'

// Provenance: legacy complete-chart projection source is
// web/app/src/views/DivinationView.vue.
const divinationMocks = vi.hoisted<{ casting: LiuyaoCasting; chart: LiuyaoChartResult }>(() => {
  const casting: LiuyaoCasting = {
    mode: 'auto',
    order: 'bottom_up',
    yaos: [7, 8, 9, 7, 8, 7],
    dong_yao: [3],
    casting_id: 'divination_adoption_casting',
  }

  return {
    casting,
    chart: {
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
      yong_shen: { name: '妻财', position: 3 },
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
      casting,
    },
  }
})

vi.mock('@shared/domain/divination', () => ({
  divinationCastingFor: vi.fn(async () => ({ casting: divinationMocks.casting })),
  divinationChartFor: vi.fn(async () => divinationMocks.chart),
}))

const serviceMocks = vi.hoisted(() => ({
  openServiceChat: vi.fn(),
}))

vi.mock('@/composables/useServiceView', () => ({
  useServiceView: () => ({
    profiles: { list: [{ id: 'profile-1' }] },
    entitlements: { purchased: true },
    chat: {},
    openServiceChat: serviceMocks.openServiceChat,
  }),
}))

const recordMocks = vi.hoisted(() => ({
  load: vi.fn(),
  save: vi.fn(async () => undefined),
}))

vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: () => ({
    records: [{
      id: 'record-1',
      title: '记录',
      input: {},
      output: divinationMocks.chart,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }],
    error: '',
    load: recordMocks.load,
    save: recordMocks.save,
    confirmRename: vi.fn(),
    confirmRemove: vi.fn(),
  }),
}))

const chart = divinationMocks.chart
const castingContext: DivinationCastingFormValue = {
  question: '近期合作如何',
  castingMode: 'auto',
  location: {
    place: '北京',
    longitude: 116.4074,
    timezone: 'Asia/Shanghai',
    source: 'manual',
  },
}

async function mountDivinationView() {
  setActivePinia(createPinia())
  const wrapper = mount(DivinationView, {
    global: {
      plugins: [createPinia()],
      stubs: {
        DivinationCastingForm: {
          name: 'DivinationCastingFormStub',
          props: { loading: Boolean, variant: String },
          data: () => ({ castingContext }),
          emits: ['submit'],
          template: `
            <button
              type="button"
              data-testid="casting-form"
              @click="$emit('submit', castingContext)"
            >
              cast
            </button>
          `,
        },
        RecordList: {
          props: ['records', 'error'],
          emits: ['open', 'rename', 'remove'],
          template: '<div data-testid="record-list">{{ records.length }}</div>',
        },
        AppConsultationCard: {
          props: ['purchased'],
          emits: ['action'],
          template: '<div data-testid="consultation-cta">{{ purchased }}</div>',
        },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('DivinationView legacy chart adoption', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders LiuyaoChartPanel with the cast result and keeps page behavior', async () => {
    const wrapper = await mountDivinationView()

    expect(recordMocks.load).toHaveBeenCalledTimes(1)
    await wrapper.get('[data-testid="casting-form"]').trigger('click')
    await flushPromises()

    const panel = wrapper.get('.app-card--highlight .liuyao-chart-panel')
    expect(panel.get('.gua-name').text()).toBe(chart.name)
    expect(panel.text()).toContain('水火未济')
    expect(panel.text()).toContain('坎宫')
    expect(panel.text()).toContain('第3爻')
    expect(panel.text()).toContain('午月：值期：午')

    const hexagramHeadings = wrapper.findAll('h2').filter(
      heading => heading.text() === '卦象',
    )
    expect(hexagramHeadings).toHaveLength(1)
    expect(wrapper.find('[data-testid="casting-form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="record-list"]').text()).toBe('1')
    expect(wrapper.find('[data-testid="consultation-cta"]').exists()).toBe(true)

    const { divinationChartFor } = await import('@shared/domain/divination')
    expect(divinationChartFor).toHaveBeenCalledWith({
      standard_time: expect.stringMatching(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+08:00$/),
      longitude: castingContext.location?.longitude,
      casting: divinationMocks.casting,
      yongShen: '世爻',
    })
  })
})
