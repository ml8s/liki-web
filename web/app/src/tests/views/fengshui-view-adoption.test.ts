import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import FengshuiView from '@/views/FengshuiView.vue'
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import BazhaiLayoutPanel from '@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'
import XuankongChartPanel from '@shared/components/engine/fengshui/XuankongChartPanel.vue'
import type { BazhaiChart, BazhaiLayout, XuankongChart } from '@shared/api/engine/fengshui'
import type { ConsultationContext } from '@shared/domain/consultation'
import { saveConsultationHandoff } from '@shared/utils/handoff'
import { LUOSHU_ORDER } from '@shared/utils/bazhai'
import { clearEngineResultCache } from '@shared/composables/useEngineResult'

const domainMocks = vi.hoisted(() => ({
  bazhaiChartFor: vi.fn(),
  bazhaiLayoutFor: vi.fn(),
  xuankongChartFor: vi.fn(),
}))

const serviceMocks = vi.hoisted(() => ({
  openServiceChat: vi.fn(),
  routerReplace: vi.fn(),
}))

const recordMocks = vi.hoisted(() => ({
  load: vi.fn(),
  save: vi.fn(),
  confirmRename: vi.fn(),
  confirmRemove: vi.fn(),
}))

vi.mock('@shared/domain/fengshui', () => ({
  bazhaiChartFor: domainMocks.bazhaiChartFor,
  bazhaiLayoutFor: domainMocks.bazhaiLayoutFor,
  xuankongChartFor: domainMocks.xuankongChartFor,
}))

vi.mock('@/composables/useServiceView', () => ({
  useServiceView: () => ({
    profiles: {
      list: [{
        id: 'profile-1',
        nickname: '张三',
        birth: {
          date: '2000-01-01',
          time: '12:00',
          instant: '2000-01-01T04:00:00Z',
          location: { longitude: 121.47, timezone: 'Asia/Shanghai' },
          gender: 'female',
        },
      }],
    },
    entitlements: { purchased: true },
    chat: { pendingIntent: '' },
    openServiceChat: serviceMocks.openServiceChat,
  }),
}))

vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: () => ({
    records: [],
    error: '',
    load: recordMocks.load,
    save: recordMocks.save,
    confirmRename: recordMocks.confirmRename,
    confirmRemove: recordMocks.confirmRemove,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { profile: 'profile-1' } }),
  useRouter: () => ({ replace: serviceMocks.routerReplace }),
}))

// Fixture provenance:
// - MingGuaSection.vue and LiuNianFlySection.vue: natal Gua, direction grid, and annual projection.
// - FengshuiView.vue: bazhai_chart is the base result and owns optional annual-result display.
const bazhaiResult: BazhaiChart = {
  ming_gua: {
    gua: { name: '艮', wuxing: '土' },
    group: '西四命',
  },
  ba_zhai_dirs: {
    sheng_qi: '西南',
    tian_yi: '西北',
    yan_nian: '西',
    fu_wei: '东北',
    huo_hai: '南',
    wu_gui: '北',
    liu_sha: '东',
    jue_ming: '东南',
  },
  liu_nian_xing: {
    year: 2026,
    ru_zhong: '七',
    gong_wei: LUOSHU_ORDER.map(gong_num => ({
      gong_num,
      xing_name: `星${gong_num}`,
    })),
  },
}

// Fixture provenance: XuankongPanel.vue period/palace/facing projection, loaded by FengshuiView.vue.
const xuankongResult: XuankongChart = {
  yun: {
    year: 2026,
    yuan: '下元',
    yun_number: 9,
    yun_name: '九运',
    start_year: 2024,
    end_year: 2043,
  },
  gong_wei: LUOSHU_ORDER.map(gong_num => ({
    gong_num,
    yun_xing: { number: 9, name: `运星${gong_num}` },
    shan_xing: { number: gong_num, name: `山星${gong_num}` },
    xiang_xing: { number: 9, name: `向星${gong_num}` },
  })),
  four_situation: {
    name: '双星会坐',
    sit_mountain_timely: true,
    face_facing_timely: false,
    sit_facing_timely: true,
    face_mountain_timely: false,
  },
  zuo_shan: 0,
  zuo_shan_name: '子',
  xiang_shan: 12,
  xiang_shan_name: '午',
  shou_shan_chu_sha: {
    zheng_shen: 9,
    ling_shen: 1,
    shou_shan: true,
    chu_sha: false,
  },
}

// Fixture provenance: BazhaiLayoutPanel.vue door/master/stove verdicts, requested only by FengshuiView.vue.
const layoutResult: BazhaiLayout = {
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
    match: '吉',
  },
  stove: {
    direction: '西北',
    gua_name: '乾',
    youxing: '天医',
    rating: '吉',
    group: '西四卦',
    match: '吉',
  },
}

const consultationLayout: ConsultationContext = {
  version: 1,
  intent: 'fengshui',
  layout: {
    zuo_shan: 0,
    period_date: '2026-01-01',
    door_gua: '坤',
    master_gua: '兑',
    stove_gua: '乾',
  },
}

async function mountView() {
  const wrapper = mount(FengshuiView, {
    attachTo: document.body,
    global: {
      stubs: {
        ServicePageShell: { template: '<section><slot /></section>' },
        FengshuiLayoutFields: true,
        AppConsultationCard: true,
        RecordList: true,
      },
    },
  })

  await flushPromises()
  await nextTick()
  await flushPromises()
  await nextTick()
  await flushPromises()
  return wrapper
}

describe('FengshuiView engine adoption', () => {
  beforeEach(() => {
    clearEngineResultCache()
    vi.clearAllMocks()
    document.body.innerHTML = ''
    const pinia = createPinia()
    setActivePinia(pinia)
    saveConsultationHandoff(consultationLayout)
  })

  it('uses the three typed engine families and FengshuiView orchestration provenance', async () => {
    domainMocks.bazhaiChartFor.mockResolvedValue(bazhaiResult)
    domainMocks.xuankongChartFor.mockResolvedValue(xuankongResult)
    domainMocks.bazhaiLayoutFor.mockResolvedValue(layoutResult)

    const wrapper = await mountView()

    const chartPanel = wrapper.findComponent(BazhaiChartPanel)
    const xuankongPanel = wrapper.findComponent(XuankongChartPanel)
    const layoutPanel = wrapper.findComponent(BazhaiLayoutPanel)

    expect(chartPanel.props('result')).toStrictEqual(bazhaiResult)
    expect(xuankongPanel.props('result')).toStrictEqual(xuankongResult)
    expect(layoutPanel.props('result')).toStrictEqual(layoutResult)
    expect(chartPanel.props('layout')).toStrictEqual(layoutResult)

    expect(domainMocks.bazhaiChartFor).toHaveBeenCalledWith('2000-01-01T12:00:00+08:00', 'female')
    expect(domainMocks.xuankongChartFor).toHaveBeenCalledWith('2026-01-01T12:00:00+08:00', 0)
    expect(domainMocks.bazhaiLayoutFor).toHaveBeenCalledWith('艮', '坤', '兑', '乾')

    expect(wrapper.find('.bazhai-chart-panel .mingua').text()).toBe('艮')
    expect(wrapper.find('.bazhai-chart-panel .sec-title').exists()).toBe(true)
    expect(wrapper.find('.xuankong-panel .xk-situation').text()).toBe('双星会坐')
    expect(wrapper.find('.bazhai-layout-panel .bz-gua-name').text()).toBe('艮')
    expect(wrapper.findAll('.bazhai-chart-panel .bz-grid .bz-cell')).toHaveLength(9)
    expect(wrapper.findAll('.bazhai-layout-panel tbody tr')).toHaveLength(3)

    wrapper.unmount()
  })

  it('keeps Xuankong when the Bazhai chart fails', async () => {
    domainMocks.bazhaiChartFor.mockRejectedValueOnce(
      Object.assign(new Error('boom'), { code: 'engine_unavailable' }),
    )
    domainMocks.xuankongChartFor.mockResolvedValue(xuankongResult)

    const wrapper = await mountView()

    expect(wrapper.findComponent(BazhaiChartPanel).exists()).toBe(false)
    expect(wrapper.findComponent(XuankongChartPanel).props('result')).toStrictEqual(xuankongResult)
    expect(wrapper.text()).toContain('排盘服务暂时不可用')
    wrapper.unmount()
  })

  it('keeps missing XuankongPanel.vue and BazhaiLayoutPanel.vue results absent', async () => {
    domainMocks.bazhaiChartFor.mockResolvedValue(bazhaiResult)
    domainMocks.xuankongChartFor.mockRejectedValue(new Error('xuankong unavailable'))
    domainMocks.bazhaiLayoutFor.mockRejectedValue(new Error('layout unavailable'))

    const wrapper = await mountView()

    expect(wrapper.findComponent(BazhaiChartPanel).props('result')).toStrictEqual(bazhaiResult)
    expect(wrapper.findComponent(XuankongChartPanel).exists()).toBe(false)
    expect(wrapper.findComponent(BazhaiLayoutPanel).exists()).toBe(false)
    expect(wrapper.find('.xuankong-panel').exists()).toBe(false)
    expect(wrapper.find('.bazhai-layout-panel').exists()).toBe(false)

    wrapper.unmount()
  })
})
