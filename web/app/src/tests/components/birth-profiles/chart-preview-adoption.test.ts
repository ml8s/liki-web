import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ChartPreview from '@/components/birth-profiles/ChartPreview.vue'
import type { NatalChart } from '@shared/domain/natal'
import type {
  BaziChartData,
  BaziFullChartData,
  BaziLiuNianOutput,
} from '@shared/api/engine/bazi'
import type { ZiweiChart, ZiweiDaXianItem, ZiweiLiuNianData } from '@shared/api/engine/ziwei'
import type { BazhaiChart } from '@shared/api/engine/fengshui'
import type { BirthProfile } from '@/api/birth-profiles'

// Fixture provenance — legacy compositions adopted by this test:
// web/shared/components/charts/BaziPanel.vue (basic/full Bazi),
// web/shared/components/charts/bazi/BaziLiunianDetail.vue,
// web/shared/components/charts/ZiweiPanel.vue,
// web/shared/components/charts/MingGuaSection.vue, and
// web/shared/components/charts/LiuNianFlySection.vue.
const domainMocks = vi.hoisted(() => ({
  natalChartForChartInput: vi.fn(),
  baziLiunianFor: vi.fn(),
  ziweiLiunianFor: vi.fn(),
  bazhaiChartFor: vi.fn(),
}))

vi.mock('@shared/domain/natal', async importOriginal => ({
  ...await importOriginal(),
  natalChartForChartInput: domainMocks.natalChartForChartInput,
  baziLiunianFor: domainMocks.baziLiunianFor,
  ziweiLiunianFor: domainMocks.ziweiLiunianFor,
}))

vi.mock('@shared/domain/fengshui', async importOriginal => ({
  ...await importOriginal(),
  bazhaiChartFor: domainMocks.bazhaiChartFor,
}))

async function panelStub(name: string, testId: string, slotName?: string) {
  const { defineComponent, h } = await import('vue')

  return defineComponent({
    name,
    props: {
      result: { type: null, required: true },
    },
    setup: (props, { slots }) => () => h(
      'div',
      { 'data-testid': testId },
      slotName ? slots[slotName]?.() : String(Boolean(props.result)),
    ),
  })
}

vi.mock('@shared/components/engine/bazi/BaziChartPanel.vue', async () => ({
  default: await panelStub('AdoptedBaziChart', 'bazi-chart'),
}))
vi.mock('@shared/components/engine/bazi/BaziFullChartPanel.vue', async () => ({
  default: await panelStub('AdoptedBaziFullChart', 'bazi-full-chart', 'before-yongshen'),
}))
vi.mock('@shared/components/engine/bazi/BaziLiuNianPanel.vue', async () => ({
  default: await panelStub('AdoptedBaziLiunian', 'bazi-liunian'),
}))
vi.mock('@shared/components/engine/ziwei/ZiweiChartPanel.vue', async () => ({
  default: await panelStub('AdoptedZiweiChart', 'ziwei-chart'),
}))
vi.mock('@shared/components/engine/ziwei/ZiweiDaXianPanel.vue', async () => ({
  default: await panelStub('AdoptedZiweiDaXian', 'ziwei-daxian'),
}))
vi.mock('@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue', async () => ({
  default: await panelStub('AdoptedZiweiLiunian', 'ziwei-liunian'),
}))
vi.mock('@shared/components/engine/fengshui/BazhaiChartPanel.vue', async () => ({
  default: await panelStub('AdoptedBazhaiChart', 'bazhai-chart'),
}))

const baziChart = {
  nian: { gan: '庚', zhi: '午', na_yin: '路旁土' },
  yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1990,
  gender: 'female',
  zi_shi_rule: '晚子时属本日',
} satisfies BaziChartData

const baziFull = {
  ...baziChart,
  nian: { ...baziChart.nian, shen_sha: [{ name: '天乙贵人' }] },
  fu_yi: { yong: '木', xi: '水', ji: '土' },
} satisfies BaziFullChartData

const baziLiuNian = {
  year: 2026,
  nian_name: '丙午',
  shi_shen: '偏财',
} satisfies BaziLiuNianOutput

const ziweiBase = {
  gong_wei: [{ index: '命宫', name: '命宫', zhi: '子', xing_yao: [] }],
  ming_gong: '命宫',
} satisfies ZiweiChart

const ziwei = {
  ...ziweiBase,
  ju_shu: '水二局',
} satisfies ZiweiChart

const ziweiDaXian = [{
  name: '甲子',
  gong: '命宫',
  qi_sui: 4,
  zhi_sui: 13,
  start_year: 1993,
  end_year: 2002,
} satisfies ZiweiDaXianItem]

const ziweiLiuNian = {
  ming_gong: '命宫',
  ming_gong_name: '兄弟',
  zhi: '子',
  si_hua: { 紫微: '禄' },
} satisfies ZiweiLiuNianData

const bazhai = {
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
    gong_wei: [{ gong_num: 4, xing_name: '文曲' }],
  },
} satisfies BazhaiChart

const profile: BirthProfile = {
  id: 'profile-1',
  user_id: 'user-1',
  version: 1,
  nickname: '测试档案',
  name: '测试档案',
  birth: {
    gender: 'female',
    date: '1990-06-15',
    time: '10:30',
    instant: '1990-06-15T02:30:00Z',
    location: {
      place: '上海',
      longitude: 121.47,
      timezone: 'Asia/Shanghai',
    },
  },
  note: '',
  third_party_consent: false,
  last_used_at: null,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const natalResult: NatalChart = {
  input: {
    instant: profile.birth.instant!,
    timezone: profile.birth.location.timezone!,
    longitude: profile.birth.location.longitude!,
  },
  bazi: baziChart,
  baziDetail: baziFull,
  ziwei,
  ziweiBase,
  ziweiDaXian,
}

function panel(wrapper: Awaited<ReturnType<typeof mountChartPreview>>, testId: string) {
  return wrapper.find(`[data-testid="${testId}"]`)
}

async function mountChartPreview(query: Record<string, string> = {}) {
  const router: Router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  await router.push({ path: '/', query })
  await router.isReady()

  const wrapper = mount(ChartPreview, {
    props: { profile, auto: true },
    global: { plugins: [router] },
  })
  await flushPromises()
  return wrapper
}

async function changeYear(wrapper: Awaited<ReturnType<typeof mountChartPreview>>, year: number) {
  const input = wrapper.findAllComponents({ name: 'ElInputNumber' }).at(-1)
  expect(input).toBeDefined()
  expect(input!.props('min')).toBe(1900)
  expect(input!.props('max')).toBe(2100)
  input!.vm.$emit('update:modelValue', year)
  await flushPromises()
  input!.vm.$emit('change', year)
  await flushPromises()
}

beforeEach(() => {
  vi.clearAllMocks()
  domainMocks.natalChartForChartInput.mockResolvedValue(natalResult)
  domainMocks.baziLiunianFor.mockResolvedValue(baziLiuNian)
  domainMocks.ziweiLiunianFor.mockResolvedValue(ziweiLiuNian)
  domainMocks.bazhaiChartFor.mockResolvedValue(bazhai)
})

describe('ChartPreview engine adoption', () => {
  it('prefers BaziFullChartPanel when bazi_fullchart exists', async () => {
    const wrapper = await mountChartPreview()

    expect(panel(wrapper, 'bazi-full-chart').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'AdoptedBaziFullChart' }).props('result')).toEqual(baziFull)
    expect(panel(wrapper, 'bazi-chart').exists()).toBe(false)
    expect(domainMocks.natalChartForChartInput).toHaveBeenCalledWith({
      instant: '1990-06-15T02:30:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
  })

  it('falls back to BaziChartPanel when only bazi_chart exists', async () => {
    domainMocks.natalChartForChartInput.mockResolvedValueOnce({
      ...natalResult,
      baziDetail: null,
    } as unknown as NatalChart)
    const wrapper = await mountChartPreview()

    expect(panel(wrapper, 'bazi-chart').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'AdoptedBaziChart' }).props('result')).toEqual(baziChart)
    expect(panel(wrapper, 'bazi-full-chart').exists()).toBe(false)
  })

  it('passes the separately loaded Bazi annual result to BaziLiuNianPanel', async () => {
    const wrapper = await mountChartPreview()

    expect(panel(wrapper, 'bazi-liunian').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'AdoptedBaziLiunian' }).props('result')).toEqual(baziLiuNian)
    expect(domainMocks.baziLiunianFor).toHaveBeenCalledWith(baziChart, new Date().getFullYear())
  })

  it('composes Ziwei chart, daxian, and host-orchestrated liunian', async () => {
    const wrapper = await mountChartPreview({ tab: 'ziwei' })

    expect(wrapper.findComponent({ name: 'AdoptedZiweiChart' }).props('result')).toEqual(ziwei)
    expect(wrapper.findComponent({ name: 'AdoptedZiweiDaXian' }).props('result')).toEqual(ziweiDaXian)
    expect(wrapper.findComponent({ name: 'AdoptedZiweiLiunian' }).props('result')).toEqual(ziweiLiuNian)
    expect(wrapper.find('.chart-year-picker').exists()).toBe(true)
    expect(domainMocks.ziweiLiunianFor).toHaveBeenCalledWith(ziweiBase, new Date().getFullYear())
  })

  it('keeps Ziwei annual loading and year bounds in ChartPreview', async () => {
    const wrapper = await mountChartPreview({ tab: 'ziwei' })
    const updatedLiuNian = { ...ziweiLiuNian, ming_gong_name: '夫妻' }
    domainMocks.ziweiLiunianFor.mockResolvedValueOnce(updatedLiuNian)

    await changeYear(wrapper, 2031)

    expect(domainMocks.ziweiLiunianFor).toHaveBeenLastCalledWith(ziweiBase, 2031)
    expect(wrapper.findComponent({ name: 'AdoptedZiweiLiunian' }).props('result')).toEqual(updatedLiuNian)
  })

  it('omits ZiweiDaXianPanel for an empty result without removing the natal chart', async () => {
    domainMocks.natalChartForChartInput.mockResolvedValueOnce({
      ...natalResult,
      ziweiDaXian: [],
    })
    const wrapper = await mountChartPreview({ tab: 'ziwei' })

    expect(panel(wrapper, 'ziwei-chart').exists()).toBe(true)
    expect(panel(wrapper, 'ziwei-daxian').exists()).toBe(false)
  })

  it('passes the complete Bazhai result family to BazhaiChartPanel', async () => {
    const wrapper = await mountChartPreview({ tab: 'bazhai' })

    expect(wrapper.findComponent({ name: 'AdoptedBazhaiChart' }).props('result')).toEqual(bazhai)
    expect(domainMocks.bazhaiChartFor).toHaveBeenCalledWith(1990, 'female')
    expect(panel(wrapper, 'bazhai-chart').exists()).toBe(true)
  })
})
