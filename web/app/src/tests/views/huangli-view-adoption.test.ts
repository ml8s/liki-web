import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import HuangliDaysPanel from '@shared/components/engine/huangli/HuangliDaysPanel.vue'
import HuangliRangeFields from '@shared/components/huangli/HuangliRangeFields.vue'
import HuangliView from '@/views/HuangliView.vue'
import type { ToolRecord } from '@/api/records'
import type { HuangliDay } from '@shared/api/engine/huangli'
import { clearEngineResultCache } from '@shared/composables/useEngineResult'

// Provenance: web/shared/components/charts/HuangliDaysPanel.vue was extracted
// into the huangli_days engine family; this test adopts it in the legacy
// HuangliView.vue theme without changing the page-level form or records flow.

const huangliViewDays = [
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
] satisfies HuangliDay[]

const huangliRecord: ToolRecord = {
  id: 'record-1',
  kind: 'huangli',
  title: '嫁娶 · 2026-06-06',
  input: { event: 'wedding', start_date: '2026-06-06', end_date: '2026-06-06' },
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const mocks = vi.hoisted(() => ({
  huangliDaysFor: vi.fn(),
  openServiceChat: vi.fn(),
}))

vi.mock('@shared/domain/huangli', async importOriginal => ({
  ...await importOriginal(),
  huangliDaysFor: mocks.huangliDaysFor,
}))

vi.mock('@/api/records', async importOriginal => ({
  ...await importOriginal(),
  fetchRecords: vi.fn(async () => [huangliRecord]),
}))

vi.mock('@/composables/useServiceView', () => ({
  useServiceView: () => ({
    profiles: { list: [{ id: 'profile-1' }] },
    entitlements: { purchased: true },
    chat: {},
    openServiceChat: mocks.openServiceChat,
  }),
}))

async function createTestRouter(): Promise<Router> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/chat', component: { template: '<div />' } },
      { path: '/birth-profiles/new', component: { template: '<div />' } },
      { path: '/', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()
  return router
}

async function mountView() {
  const router = await createTestRouter()
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(HuangliView, {
    global: { plugins: [router, pinia] },
  })
  await flushPromises()
  return wrapper
}

async function submitRange(wrapper: Awaited<ReturnType<typeof mountView>>) {
  wrapper.findComponent(HuangliRangeFields).vm.$emit('update:patch', {
    startDate: '2026-06-06',
    endDate: '2026-06-06',
  })
  await flushPromises()
  await wrapper.find('form').trigger('submit')
  await flushPromises()
}

describe('HuangliView adoption — source: web/shared/components/charts/HuangliDaysPanel.vue + web/app/src/views/HuangliView.vue', () => {
  beforeEach(() => {
    clearEngineResultCache()
    vi.clearAllMocks()
  })

  it('calls the huangli domain API and passes its typed result to the engine days panel', async () => {
    mocks.huangliDaysFor.mockResolvedValueOnce(huangliViewDays)
    const wrapper = await mountView()

    await submitRange(wrapper)

    expect(mocks.huangliDaysFor).toHaveBeenCalledWith('wedding', '2026-06-06', 1)
    const enginePanel = wrapper.findComponent(HuangliDaysPanel)
    expect(enginePanel.exists()).toBe(true)
    expect(enginePanel.props('result')).toEqual(huangliViewDays)
    expect(String(enginePanel.vm.$.type.__file)).toContain('components/engine/huangli/HuangliDaysPanel.vue')
    expect(wrapper.get('.range-info').text()).toContain('2026-06-06')
    expect(wrapper.get('.day.good').text()).toContain('2026-06-06')
    expect(wrapper.text()).toContain('黄历推荐（结婚）')
    expect(wrapper.get('.record-row').text()).toContain('嫁娶 · 2026-06-06')
    expect(wrapper.get('form').text()).toContain('保存记录')
    expect(wrapper.find('.record-list').exists()).toBe(true)

    await wrapper.get('.action button').trigger('click')
    expect(mocks.openServiceChat).toHaveBeenCalledWith(
      'profile-1',
      'huangli',
      expect.any(Function),
      {
        version: 1,
        intent: 'huangli',
        event: 'wedding',
        start_date: '2026-06-06',
        end_date: '2026-06-06',
      },
    )
    wrapper.unmount()
  })

  it('renders the engine panel empty branch without fabricated days', async () => {
    mocks.huangliDaysFor.mockResolvedValueOnce([])
    const wrapper = await mountView()

    await submitRange(wrapper)

    const enginePanel = wrapper.findComponent(HuangliDaysPanel)
    expect(enginePanel.props('result')).toEqual([])
    expect(wrapper.text()).toContain('该范围内没有推荐吉日。')
    expect(wrapper.find('.day').exists()).toBe(false)
    expect(wrapper.find('.record-row').exists()).toBe(true)
    wrapper.unmount()
  })
})
