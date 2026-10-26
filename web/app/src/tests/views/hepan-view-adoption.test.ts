import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import HepanView from '@/views/HepanView.vue'
import BaziBondPanel from '@shared/components/engine/bazi/BaziBondPanel.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import RecordList from '@/components/records/RecordList.vue'
import type { BaziBondResult, BaziChartData } from '@shared/api/engine/bazi'
import type { ToolRecord } from '@/api/records'

/**
 * Provenance: web/shared/components/charts/HepanPanel.vue and
 * web/app/src/views/HepanView.vue.
 */
const chartA: BaziChartData = {
  nian: { gan: '庚', zhi: '辰', na_yin: '白蜡金' },
  yue: { gan: '乙', zhi: '酉', na_yin: '泉中水' },
  ri: { gan: '甲', zhi: '辰', na_yin: '覆灯火' },
  shi: { gan: '丙', zhi: '子', na_yin: '涧下水' },
  birth_year: 1990,
  gender: 'male',
  zi_shi_rule: '晚子时属本日',
}
const chartB: BaziChartData = {
  ...chartA,
  nian: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  yue: { gan: '甲', zhi: '午', na_yin: '沙中金' },
  ri: { gan: '丙', zhi: '申', na_yin: '山下火' },
  shi: { gan: '戊', zhi: '戌', na_yin: '平地木' },
  birth_year: 1992,
  gender: 'female',
}
const bond: BaziBondResult = {
  day_master: {
    a: { gan: '甲', gender: 'male', wuxing: '木' },
    b: { gan: '丙', gender: 'female', wuxing: '火' },
    gan_guan_xi: { gan_a: '甲', gan_b: '丙', type: '相生', relation: '甲生丙' },
    b_present_in_a: false,
  },
  spouse_palace: {},
  zhu_cross: { pairs: [] },
  shi_shen_cross: { a_to_b: {}, b_to_a: {} },
  wuxing_cross: {},
  nayin_cross: { pairs: [] },
  spouse_star: {},
  shensha_cross: {},
}
const secondBirth = {
  gender: 'female' as const,
  date: '1992-06-18',
  time: '10:30',
  location: {
    place: '',
    longitude: 116.4,
    timezone: 'Asia/Shanghai',
    source: 'saved' as const,
  },
  third_party_consent: true,
}

const service = vi.hoisted(() => ({
  openServiceChat: vi.fn(async () => undefined),
}))
const toolRecords = vi.hoisted(() => ({
  kind: [] as string[],
  records: [] as ToolRecord[],
  save: vi.fn(async () => undefined),
  load: vi.fn(async () => undefined),
  confirmRename: vi.fn(async () => undefined),
  confirmRemove: vi.fn(async () => undefined),
}))
const hepanDomain = vi.hoisted(() => ({
  hepanForChartInputs: vi.fn(),
}))
const natalDomain = vi.hoisted(() => ({
  baziChartForChartInput: vi.fn(),
}))

vi.mock('@/composables/useServiceView', () => ({
  useServiceView: () => ({
    profiles: {
      list: [{
        id: 'person-1',
        nickname: '张三',
        birth: {
          instant: '1990-04-15T08:00:00Z',
          gender: 'male',
          location: { timezone: 'Asia/Shanghai', longitude: 116.4 },
        },
      }],
    },
    entitlements: { allowedSkills: ['liki'] },
    chat: { pendingIntent: '' },
    openServiceChat: service.openServiceChat,
  }),
}))
vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: (kind: string) => {
    toolRecords.kind.push(kind)
    return {
      records: toolRecords.records,
      error: '',
      load: toolRecords.load,
      save: toolRecords.save,
      confirmRename: toolRecords.confirmRename,
      confirmRemove: toolRecords.confirmRemove,
    }
  },
}))
vi.mock('@shared/domain/hepan', () => hepanDomain)
vi.mock('@shared/domain/natal', () => natalDomain)

async function mountView(): Promise<VueWrapper> {
  setActivePinia(createPinia())
  const wrapper = mount(HepanView, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        // Keep the adoption boundary deterministic: validation is an app-form
        // concern already covered by form-kit; this test owns result mapping.
        ElForm: {
          template: '<form><slot /></form>',
          setup: (_props, { expose }) => {
            expose({ validate: async () => true })
          },
        },
        BirthFields: { template: '<div />' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

async function completeForm(wrapper: VueWrapper) {
  const profileSelect = wrapper.findComponent({ name: 'ElSelect' })
  profileSelect.vm.$emit('update:modelValue', 'person-1')
  profileSelect.vm.$emit('change', 'person-1')
  await flushPromises()

  wrapper.findComponent(BirthFields).vm.$emit('update:patch', secondBirth)
  await wrapper.find('input[type="checkbox"]').setValue(true)
  await wrapper.findAll('form')[1]!.trigger('submit')
  await vi.waitFor(() => expect(hepanDomain.hepanForChartInputs).toHaveBeenCalled())
  await flushPromises()
}

function button(wrapper: VueWrapper, label: string) {
  const target = wrapper.findAll('button').find(node => node.text() === label)
  expect(target, `expected button ${label}`).toBeTruthy()
  return target!
}

  describe('HepanView BaziBondPanel adoption (HepanPanel.vue + HepanView.vue provenance)', () => {
    let activeWrapper: VueWrapper | null = null
      beforeEach(() => {
      localStorage.clear()
      sessionStorage.clear()
      document.body.innerHTML = ''
      vi.clearAllMocks()
      toolRecords.kind = []
      toolRecords.records = []
      natalDomain.baziChartForChartInput.mockResolvedValue(chartA)
      hepanDomain.hepanForChartInputs.mockResolvedValue({ chartA, chartB, bond })
    })

    afterEach(() => {
      activeWrapper?.unmount()
      activeWrapper = null
      document.body.innerHTML = ''
    })

    it('renders the adopted panel and preserves consent, records, and CTA boundaries', async () => {
      const record: ToolRecord = {
        id: 'record-1',
        kind: 'bond',
        title: '张三 · 1992-06-18',
        input: { profile_id: 'person-1', b: secondBirth },
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }
      toolRecords.records = [record]
      const wrapper = await mountView()
      activeWrapper = wrapper

      expect(toolRecords.kind).toEqual(['bond'])
      expect(toolRecords.load).toHaveBeenCalledTimes(1)
      expect(wrapper.findComponent(BaziBondPanel).exists()).toBe(false)

      await completeForm(wrapper)

      const panel = wrapper.findComponent(BaziBondPanel)
      expect(panel.props('result')).toEqual(bond)
      expect(panel.props('chartA')).toEqual(chartA)
      expect(panel.props('chartB')).toEqual(chartB)
      expect(panel.props('nameA')).toBe('张三')
      expect(panel.props('nameB')).toBeUndefined()

      await wrapper.find('input[type="checkbox"]').setValue(false)
      await flushPromises()
      expect(wrapper.findComponent(BaziBondPanel).exists()).toBe(false)
      expect(wrapper.findAll('button').some(node => node.text() === '去合盘')).toBe(false)

      await wrapper.find('input[type="checkbox"]').setValue(true)
      wrapper.findComponent(RecordList).vm.$emit('open', record)
      await flushPromises()
      expect(wrapper.findComponent(BaziBondPanel).exists()).toBe(true)
      await button(wrapper, '保存记录').trigger('click')
      expect(toolRecords.save).toHaveBeenCalledWith('合盘 · 张三 × 1992-06-18', {
        profile_id: 'person-1',
        b: expect.objectContaining(secondBirth),
        third_party_consent: true,
      })

      await button(wrapper, '去合盘').trigger('click')
      await flushPromises()
      expect(service.openServiceChat).toHaveBeenCalledWith(
        'person-1',
        'hepan',
        expect.any(Function),
        expect.objectContaining({
          scenario: 'hepan',
          third_party_consent: true,
          second_birth: expect.objectContaining({ date: secondBirth.date }),
        }),
      )
      wrapper.unmount()
    })

    it('does not render or fabricate the panel when either source chart is absent', async () => {
      hepanDomain.hepanForChartInputs.mockResolvedValue({
        chartA: null,
        chartB: null,
        bond,
      })
      const wrapper = await mountView()
      activeWrapper = wrapper
      await completeForm(wrapper)

      expect(hepanDomain.hepanForChartInputs).toHaveBeenCalledTimes(1)
      expect(wrapper.findComponent(BaziBondPanel).exists()).toBe(false)
      wrapper.unmount()
    })
  })
