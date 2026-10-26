import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import DomainFlowView from '@/views/DomainFlowView.vue'
import FlowDivinationForm from '@/components/flow/forms/FlowDivinationForm.vue'
import ArtifactSurface from '@/components/flow/artifacts/ArtifactSurface.vue'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useChatStore } from '@/stores/chat'
import { useDomainFlowStore } from '@/stores/domain-flow'
import { useEntitlementsStore } from '@/stores/entitlements'
import type { ChatSession } from '@/api/chat'
import type { BirthProfile } from '@/api/birth-profiles'
import FlowHepanForm from '@/components/flow/forms/FlowHepanForm.vue'
import FlowFengshuiForm from '@/components/flow/forms/FlowFengshuiForm.vue'
import FlowNamingForm from '@/components/flow/forms/FlowNamingForm.vue'
import FlowHistoryPanel from '@/components/flow/FlowHistoryPanel.vue'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import { useSalesStore } from '@/stores/sales-store'
import type { BaziBondResult, BaziChartData } from '@shared/api/engine/bazi'

const mediaQuery = vi.hoisted(() => ({ matches: false }))

 vi.mock('@vueuse/core', async () => {
   const { ref } = await import('vue')
   return {
     useMediaQuery: () => ref(mediaQuery.matches),
   }
 })

function birthProfile(id = 'person-1', nickname = '张三'): BirthProfile {
  return {
    id,
    user_id: 'user-1',
    version: 1,
    nickname,
    name: '',
    birth: {
      gender: 'female',
      date: '2000-01-01',
      time: '12:00',
      location: { place: '', longitude: 116.4, timezone: 'Asia/Shanghai' },
      instant: null,
    },
    note: '',
    third_party_consent: false,
    last_used_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }
}

function chatSession(overrides: Partial<ChatSession> = {}): ChatSession {
  return {
    id: 'session-1',
    user_id: 'user-1',
    profile_id: 'person-1',
    skill: 'liki',
    expert: 'natal',
    title: '张三 · 命理解读',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
    last_message_at: null,
    ...overrides,
  }
}

function divinationSession(): ChatSession {
  return chatSession({
    id: 'divination-session',
    skill: 'divination',
    expert: 'divination',
    title: '问卦解读',
    updated_at: '2026-01-04T00:00:00Z',
    context: {
      version: 1,
      scenario: 'divination',
      question: '近期合作如何',
      casting: {
        mode: 'auto',
        order: 'bottom_up',
        yaos: [7, 8, 8, 7, 8, 7],
        dong_yao: [],
        casting_id: 'a'.repeat(64),
      },
    },
  })
}

async function createTestRouter(initialUrl = '/') {
  const router = createRouter({
    history: createMemoryHistory('/app/'),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div />' } },
      { path: '/consultation', name: 'app-consultation', component: { template: '<div />' } },
      { path: '/me', name: 'app-me', component: { template: '<div />' } },
      { path: '/birth-profiles', name: 'birth-profiles', component: { template: '<div />' } },
      { path: '/birth-profiles/new', name: 'birth-profile-new', component: { template: '<div />' } },
    ],
  })
  router.push(initialUrl)
  await router.isReady()
  return router
}

async function flushAll() {
  await flushPromises()
  await vi.dynamicImportSettled?.()
  await flushPromises()
}

async function settleTransitions() {
  for (let index = 0; index < 3; index += 1) {
    await nextTick()
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, 125))
  }
}

async function mountWithSessions(sessions: ChatSession[]) {
  const profiles = useBirthProfilesStore()
  const chat = useChatStore()
  profiles.list = [birthProfile()]
  chat.sessions = sessions
  vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
  vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
  vi.spyOn(chat, 'openSession').mockImplementation(async sessionId => {
    chat.activeSession = chat.sessions.find(session => session.id === sessionId) ?? null
  })
  vi.spyOn(useEntitlementsStore(), 'load').mockResolvedValue(undefined)

  const wrapper = mount(DomainFlowView, {
    attachTo: document.body,
    global: { plugins: [await createTestRouter()] },
  })
  await flushAll()
  return wrapper
}

const divinationMocks = vi.hoisted(() => {
  const casting = {
    mode: 'auto' as const,
    order: 'bottom_up' as const,
    yaos: [7, 8, 8, 7, 8, 7] as [7, 8, 8, 7, 8, 7],
    dong_yao: [3] as [3],
    casting_id: 'casting_test',
  }
  const chart = {
    name: '火雷噬嗑',
    ben_gua: '震为雷',
    gong: '震宫',
    gong_wuxing: '木',
    lines: [],
    bian_yao: [],
    ri_chen_gan: '甲',
    ri_chen_zhi: '子',
    yue_jian_zhi: '寅',
    yong_shen: { name: '妻财' },
    gua_ci: { name: '火雷噬嗑', gua_ci: '利用狱。', yao_ci: ['', '', '', '', '', ''] },
    wang_shuai: ['', '', '', '', '', ''] as [string, string, string, string, string, string],
    ri_chen_relations: ['', '', '', '', '', ''] as [unknown, unknown, unknown, unknown, unknown, unknown],
    xun_kong: ['戌', '亥'] as [string, string],
    casting,
  }
  return { casting, chart }
})

vi.mock('@shared/domain/divination', async importOriginal => ({
  ...await importOriginal(),
  divinationCastingFor: vi.fn(async () => ({ casting: divinationMocks.casting })),
  divinationChartFor: vi.fn(async () => divinationMocks.chart),
}))

const saveToolRecord = vi.fn(async () => undefined)
const saveBondRecord = vi.fn(async () => undefined)
const saveBazhaiRecord = vi.fn(async () => undefined)
const namingMocks = vi.hoisted(() => ({
  chart: {} as import('@shared/api/engine/bazi').BaziChartData,
  assessment: {} as import('@shared/api/engine/bazi').BaziYongShenResult,
}))
const hepanMocks = vi.hoisted(() => {
  const chart = {} as BaziChartData
  const bond = {} as BaziBondResult
  return {
    chartA: chart,
    chartB: { ...chart } as BaziChartData,
    bond,
  }
})

const fengshuiMocks = vi.hoisted(() => {
  const bazhai = {
    ming_gua: { gua: { name: '艮', wuxing: '土' }, group: '西四命' },
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
  }
  const layout = {
    group: '西四命',
    ming_gua: '艮',
    door: { direction: '西南', gua_name: '坤', youxing: '生气', rating: '大吉', group: '西四卦', match: '吉' },
    master: { direction: '西', gua_name: '兑', youxing: '延年', rating: '吉', group: '西四卦', match: '吉' },
    stove: { direction: '西北', gua_name: '乾', youxing: '天医', rating: '吉', group: '西四卦', match: '吉' },
  }
  const xuankong = {
    yun: { year: 2026, yun_number: 9, yun_name: '九运', start_year: 2024, end_year: 2043 },
    gong_wei: Array.from({ length: 9 }, (_, index) => ({
      gong_num: index + 1,
      yun_xing: { number: 5, name: '五黄廉贞' },
      shan_xing: { number: 9, name: '九紫右弼' },
      xiang_xing: { number: 9, name: '九紫右弼' },
    })),
    four_situation: { name: '双星会坐', sit_mountain_timely: true, face_facing_timely: false, sit_facing_timely: true, face_mountain_timely: false },
    zuo_shan: 0,
    zuo_shan_name: '子',
    xiang_shan: 12,
    xiang_shan_name: '午',
    shou_shan_chu_sha: { zheng_shen: 9, ling_shen: 1, shou_shan: true, chu_sha: false },
  }
  return { bazhai, layout, xuankong }
})

vi.mock('@shared/domain/natal', async importOriginal => ({
  ...await importOriginal(),
  baziChartFor: vi.fn(async () => namingMocks.chart),
}))

vi.mock('@shared/domain/naming', async importOriginal => ({
  ...await importOriginal(),
  namingAssessmentFor: vi.fn(async () => namingMocks.assessment),
}))

vi.mock('@shared/domain/fengshui', async importOriginal => ({
  ...await importOriginal(),
  bazhaiChartFor: vi.fn(async () => fengshuiMocks.bazhai),
  bazhaiLayoutFor: vi.fn(async () => fengshuiMocks.layout),
  xuankongChartFor: vi.fn(async () => fengshuiMocks.xuankong),
}))

vi.mock('@shared/domain/hepan', async importOriginal => ({
  ...await importOriginal(),
  hepanForChartInputs: vi.fn(async () => hepanMocks),
}))

vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: (kind: string) => ({ save: kind === 'bond' ? saveBondRecord : kind === 'bazhai' ? saveBazhaiRecord : saveToolRecord }),
}))

describe('DomainFlowView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
    vi.clearAllMocks()
    mediaQuery.matches = false
    document.body.innerHTML = ''
  })

  it('restores the latest visible session for the selected person/domain', async () => {
    const profiles = useBirthProfilesStore()
    const chat = useChatStore()
    profiles.list = [birthProfile()]
    chat.sessions = [
      chatSession({
        id: 'older-natal',
        updated_at: '2026-01-01T00:00:00Z',
      }),
      chatSession({
        id: 'latest-natal',
        updated_at: '2026-01-03T00:00:00Z',
      }),
      chatSession({
        id: 'divination-domain',
        expert: 'divination',
        profile_id: '',
        updated_at: '2026-01-04T00:00:00Z',
      }),
    ]
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    vi.spyOn(chat, 'openSession').mockImplementation(async sessionId => {
      chat.activeSession = chat.sessions.find(session => session.id === sessionId) ?? null
    })
    vi.spyOn(useEntitlementsStore(), 'load').mockResolvedValue(undefined)

    const router = await createTestRouter()
    const wrapper = mount(DomainFlowView, {
      global: { plugins: [router] },
    })
    await flushAll()

    const flow = useDomainFlowStore()
    expect(flow.personId).toBe('person-1')
    expect(flow.domainId).toBe('natal')
    expect(chat.activeSession?.id).toBe('latest-natal')
    expect(wrapper.find('.flow-restore').exists()).toBe(false)
    wrapper.unmount()
  })

  it('creates a new natal session with the domain base skill', async () => {
    const profiles = useBirthProfilesStore()
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    profiles.list = [birthProfile()]
    entitlements.allowedSkills = ['liki']
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession').mockResolvedValue(chatSession())
    const send = vi.spyOn(chat, 'send').mockResolvedValue(undefined)

    const router = await createTestRouter('/')
    const wrapper = mount(DomainFlowView, {
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const flow = useDomainFlowStore()
    flow.setDraft('今年事业怎么样')
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(createSession).toHaveBeenCalledWith('person-1', 'natal')
    expect(chat.activeSession?.skill).toBe('liki')
    expect(send).toHaveBeenCalledWith('今年事业怎么样')
    expect(flow.getDraft()).toBe('')
    wrapper.unmount()
  })

  it('keeps huangli pending instead of routing it to the wrong expert', async () => {
    const profiles = useBirthProfilesStore()
    const chat = useChatStore()
    const entitlements = useEntitlementsStore()
    profiles.list = [birthProfile()]
    entitlements.allowedSkills = ['liki']
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession')

    const router = await createTestRouter('/?seed=huangli')
    const wrapper = mount(DomainFlowView, {
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const flow = useDomainFlowStore()
    flow.setDraft('下个月哪天结婚好')
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.flow-error').text()).toContain('迁移到命理专家')
    expect(createSession).not.toHaveBeenCalled()
    expect(flow.getDraft()).toBe('下个月哪天结婚好')
    wrapper.unmount()
  })

  it('renders no artifact capsule without a deterministic payload', async () => {
    const wrapper = await mountWithSessions([chatSession()])

    expect(wrapper.find('.artifact-capsule').exists()).toBe(false)
    expect(document.body.querySelector('.liuyao-panel')).toBeNull()
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('renders a real casting receipt and isolates artifact state by domain', async () => {
    const wrapper = await mountWithSessions([
      chatSession({ id: 'natal-session' }),
      divinationSession(),
    ])
    const flow = useDomainFlowStore()

    flow.setDomain('divination')
    await flushAll()
    const capsule = wrapper.get('.artifact-capsule')
    expect(capsule.text()).toContain('问卦')
    expect(capsule.attributes('aria-expanded')).toBe('false')

    await capsule.trigger('click')
    await flushAll()
    expect(capsule.attributes('aria-expanded')).toBe('true')
    expect(document.body.textContent).toContain('火雷噬嗑')

    flow.setDomain('natal')
    await flushAll()
    expect(wrapper.find('.artifact-capsule').exists()).toBe(false)
    expect(document.body.querySelector('.liuyao-panel')).toBeNull()

    flow.setDomain('divination')
    await flushAll()
    expect(wrapper.get('.artifact-capsule').attributes('aria-expanded')).toBe('true')
    expect(document.body.querySelector('.liuyao-panel')).not.toBeNull()
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('closes the mobile artifact sheet and restores capsule focus', async () => {
    const wrapper = await mountWithSessions([divinationSession()])
    const flow = useDomainFlowStore()

    flow.setDomain('divination')
    await flushAll()
    await wrapper.get('.artifact-capsule').trigger('click')
    await flushAll()

    expect(wrapper.find('aside.artifact-sidecar').exists()).toBe(false)
    expect(document.body.querySelector('.liuyao-panel')).not.toBeNull()

    const closeButtons = [...document.querySelectorAll<HTMLButtonElement>('.flow-sheet__close')]
    closeButtons.at(-1)!.click()
    await settleTransitions()
    expect(wrapper.get('.artifact-capsule').attributes('aria-expanded')).toBe('false')
    expect(wrapper.getComponent(ArtifactSurface).props('open')).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('.artifact-capsule').element)
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('keeps a closed desktop sidecar hidden but focus-addressable', async () => {
    mediaQuery.matches = true
    const wrapper = await mountWithSessions([divinationSession()])
    const flow = useDomainFlowStore()

    flow.setDomain('divination')
    await flushAll()
    expect(wrapper.find('aside.artifact-sidecar').exists()).toBe(false)

    await wrapper.get('.artifact-capsule').trigger('click')
    await flushAll()
    expect(wrapper.get('aside.artifact-sidecar').classes()).toContain('artifact-sidecar--peek')
    expect(wrapper.find('.artifact-layer--sidecar').exists()).toBe(true)

    await wrapper.get('.artifact-sidecar__close').trigger('click')
    await settleTransitions()
    expect(wrapper.find('aside.artifact-sidecar').exists()).toBe(true)
    expect(wrapper.find('aside.artifact-sidecar').isVisible()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('.artifact-capsule').element)
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('casts in the divination domain and emits a real liuyao artifact', async () => {
    const profiles = useBirthProfilesStore()
    const entitlements = useEntitlementsStore()
    const chat = useChatStore()
    const flow = useDomainFlowStore()
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)

    const router = await createTestRouter('/?domain=divination')
    const wrapper = mount(DomainFlowView, {
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const form = wrapper.findComponent(FlowDivinationForm)
    await form.vm.$.exposed?.restore({
      question: '近期合作如何',
      castingMode: 'auto',
      location: {
        place: '上海',
        longitude: 121.4726,
        timezone: 'Asia/Shanghai',
        source: 'saved',
      },
    })
    await form.find('form').trigger('submit')
    await flushPromises()

    expect(flow.domainId).toBe('divination')
    expect(wrapper.find('.artifact-capsule').exists()).toBe(true)
    expect(saveToolRecord).toHaveBeenCalledTimes(1)
    expect(divinationMocks.casting.casting_id).toBe('casting_test')
    wrapper.unmount()
  })

  it('computes hepan from a focused natal form and keeps consent in the follow-up context', async () => {
    const profiles = useBirthProfilesStore()
    const entitlements = useEntitlementsStore()
    const chat = useChatStore()
    const profile = birthProfile()
    profile.birth.instant = '2000-01-01T04:00:00Z'
    profiles.list = [profile]
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession').mockResolvedValue(
      chatSession({
        id: 'hepan-created',
        expert: 'natal',
        context: {
          version: 1,
          scenario: 'hepan',
          second_birth: {
            gender: 'female',
            date: '1992-06-18',
            time: '10:30',
            location: { longitude: 116.4, timezone: 'Asia/Shanghai' },
          },
          third_party_consent: true,
        },
      }),
    )
    const send = vi.spyOn(chat, 'send').mockResolvedValue(undefined)

    const router = await createTestRouter('/?seed=hepan')
    const wrapper = mount(DomainFlowView, {
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const form = wrapper.getComponent(FlowHepanForm)
    form.vm.$emit('submit', {
      secondBirth: {
        gender: 'female',
        date: '1992-06-18',
        time: '10:30',
        location: {
          place: '',
          longitude: 116.4,
          timezone: 'Asia/Shanghai',
          source: 'saved' as const,
        },
      },
      thirdPartyConsent: true,
    })
    await flushAll()

    expect(wrapper.find('.artifact-capsule').exists()).toBe(true)
    await wrapper.get('.flow-inline-actions button').trigger('click')
    await flushPromises()
    expect(saveBondRecord).toHaveBeenCalledTimes(1)

    await wrapper.find('textarea').setValue('帮我们解读关系节奏')
    await wrapper.find('form.flow-composer').trigger('submit')
    await flushPromises()

    expect(createSession).toHaveBeenCalledWith(
      'person-1',
      'hepan',
      expect.objectContaining({
        version: 1,
        scenario: 'hepan',
        third_party_consent: true,
        second_birth: expect.objectContaining({ date: '1992-06-18' }),
      }),
    )
    expect(send).toHaveBeenCalledWith('帮我们解读关系节奏')
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('computes fengshui artifacts and sends a fengshui-scoped follow-up', async () => {
    const profiles = useBirthProfilesStore()
    const entitlements = useEntitlementsStore()
    const chat = useChatStore()
    profiles.list = [birthProfile()]
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession').mockResolvedValue(
      chatSession({
        id: 'fengshui-created',
        expert: 'fengshui',
        context: {
          version: 1,
          scenario: 'fengshui',
          layout: {
            zuo_shan: 0,
            period_date: '2026-01-01',
            door_gua: '乾',
            master_gua: '坤',
            stove_gua: '兑',
          },
        },
      }),
    )
    const send = vi.spyOn(chat, 'send').mockResolvedValue(undefined)

    const router = await createTestRouter('/?domain=fengshui&seed=fengshui')
    const wrapper = mount(DomainFlowView, {
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const form = wrapper.getComponent(FlowFengshuiForm)
    form.vm.$emit('submit', {
      zuoShan: 0,
      periodDate: '2026-01-01',
      doorGua: '乾',
      masterGua: '坤',
      stoveGua: '兑',
    })
    await flushAll()

    expect(wrapper.find('.artifact-capsule').exists()).toBe(true)
    await wrapper.get('.flow-inline-actions button').trigger('click')
    await flushPromises()
    expect(saveBazhaiRecord).toHaveBeenCalledTimes(1)

    await wrapper.find('textarea').setValue('帮我判断客厅布局')
    await wrapper.find('form.flow-composer').trigger('submit')
    await flushPromises()

    expect(createSession).toHaveBeenCalledWith(
      'person-1',
      'fengshui',
      expect.objectContaining({
        version: 1,
        scenario: 'fengshui',
        layout: expect.objectContaining({ door_gua: '乾', master_gua: '坤', stove_gua: '兑' }),
      }),
    )
    expect(send).toHaveBeenCalledWith('帮我判断客厅布局')
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('computes a naming assessment and sends naming-scoped birth context', async () => {
    const profiles = useBirthProfilesStore()
    const entitlements = useEntitlementsStore()
    const chat = useChatStore()
    profiles.list = [birthProfile()]
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession').mockResolvedValue(
      chatSession({
        id: 'naming-created',
        expert: 'naming',
        context: {
          version: 1,
          scenario: 'naming',
          birth: {
            gender: 'female',
            date: '2000-01-01',
            time: '12:00',
            location: { longitude: 116.4, timezone: 'Asia/Shanghai' },
          },
        },
      }),
    )
    const send = vi.spyOn(chat, 'send').mockResolvedValue(undefined)

    const router = await createTestRouter('/?domain=naming&seed=naming')
    const wrapper = mount(DomainFlowView, {
      attachTo: document.body,
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    await wrapper.getComponent(FlowNamingForm).find('button').trigger('click')
    await flushAll()
    expect(wrapper.find('.artifact-capsule').exists()).toBe(true)

    await wrapper.find('textarea').setValue('帮助我选字方向')
    await wrapper.find('form.flow-composer').trigger('submit')
    await flushPromises()

    expect(createSession).toHaveBeenCalledWith(
      'person-1',
      'naming',
      expect.objectContaining({
        version: 1,
        scenario: 'naming',
        birth: expect.objectContaining({ date: '2000-01-01', time: '12:00' }),
      }),
    )
    expect(send).toHaveBeenCalledWith('帮助我选字方向')
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('shows person/domain history and restores the selected session', async () => {
    const openSession = vi.fn(async () => undefined)
    const wrapper = await mountWithSessions([
      chatSession({ id: 'same-channel-active', title: '命理当前' }),
      chatSession({ id: 'same-channel', title: '命理历史' }),
      chatSession({
        id: 'other-person',
        profile_id: 'person-2',
        title: '其他档案',
      }),
      chatSession({
        id: 'other-domain',
        expert: 'divination',
        profile_id: '',
        title: '问卦历史',
      }),
    ])
    const chat = useChatStore()
    chat.openSession = openSession
    await flushPromises()

    const history = wrapper.getComponent(FlowHistoryPanel)
    expect(history.props('sessions').map(session => session.id)).toEqual(['same-channel-active', 'same-channel'])
    expect(wrapper.text()).toContain('命理历史')

    history.vm.$emit('open', chat.sessions[1]!)
    await flushPromises()
    expect(openSession).toHaveBeenCalledWith('same-channel')
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('offers inline consultation purchase and preserves the routing seed', async () => {
    const sales = useSalesStore()
    vi.spyOn(sales, 'loadProducts').mockResolvedValue(undefined)
    sales.products = [{
      id: 'liki-chat',
      name: 'Liki Chat',
      description: 'Consultation',
      skills: ['liki'],
      display_currency: 'USD',
      display_amount_minor: 900,
      purchase_options: [],
      purchased: false,
    } satisfies import('@/api/sales').Product]
    const buy = vi.spyOn(sales, 'buy').mockResolvedValue(undefined)
    const wrapper = await mountWithSessions([])

    expect(wrapper.getComponent(ProductPurchaseCard).props('product')).toMatchObject({ id: 'liki-chat' })
    wrapper.getComponent(ProductPurchaseCard).vm.$emit('purchase', 'one_time')
    await flushPromises()
    expect(buy).toHaveBeenCalledWith('one_time', 'natal')
    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('tracks near-bottom state outside the frozen shell contract', async () => {
    const wrapper = await mountWithSessions([chatSession()])
    const chat = useChatStore()
    chat.messages = [{
      id: 'message-1',
      session_id: 'session-1',
      user_id: 'user-1',
      role: 'assistant',
      kind: 'text',
      content: '最新解读',
      created_at: '2026-01-02T00:00:00Z',
    }]
    const main = wrapper.get('.flow-main').element as HTMLElement
    Object.defineProperties(main, {
      scrollHeight: { configurable: true, value: 1_000 },
      clientHeight: { configurable: true, value: 400 },
      scrollTop: { configurable: true, value: 0, writable: true },
      scrollTo: { configurable: true, value: vi.fn() },
    })
    const focus = vi.spyOn(main, 'focus').mockImplementation(() => {})

    main.dispatchEvent(new Event('scroll'))
    await nextTick()
    expect(wrapper.find('.flow-follow').exists()).toBe(true)

    await wrapper.get('.flow-follow').trigger('click')
    await flushPromises()
    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    expect(main.scrollTo).toHaveBeenCalled()

    wrapper.unmount()
    document.body.innerHTML = ''
  })

  it('sends a follow-up question with the complete divination context', async () => {
    const chat = useChatStore()
    const send = vi.spyOn(chat, 'send').mockResolvedValue(undefined)
    const createSession = vi.spyOn(chat, 'createSession').mockResolvedValue(
      chatSession({
        id: 'divination-created',
        expert: 'divination',
        context: {
          version: 1,
          scenario: 'divination',
          question: '近期合作如何',
          casting: divinationMocks.casting,
          casting_instant: '2026-01-01T00:00:00Z',
          location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
        },
      }),
    )

    const profiles = useBirthProfilesStore()
    const entitlements = useEntitlementsStore()
    const flow = useDomainFlowStore()
    vi.spyOn(profiles, 'loadList').mockResolvedValue(undefined)
    vi.spyOn(entitlements, 'load').mockResolvedValue(undefined)
    vi.spyOn(chat, 'loadSessions').mockResolvedValue(undefined)

    flow.setPerson('person-1')
    const router = await createTestRouter('/?domain=divination')
    const wrapper = mount(DomainFlowView, {
      global: {
        plugins: [router],
        stubs: { ChatThread: true, ChatStreamStatus: true },
      },
    })
    await flushAll()

    const form = wrapper.findComponent(FlowDivinationForm)
    await form.vm.$.exposed?.restore({
      question: '近期合作如何',
      castingMode: 'auto',
      location: {
        place: '上海',
        longitude: 121.4726,
        timezone: 'Asia/Shanghai',
        source: 'saved',
      },
    })
    await form.find('form').trigger('submit')
    await flushPromises()

    await wrapper.find('textarea').setValue('这件事应该怎么处理？')
    await wrapper.find('form.flow-composer').trigger('submit')
    await flushPromises()

    expect(createSession).toHaveBeenCalledWith(
      'person-1',
      'divination',
      expect.objectContaining({
        version: 1,
        scenario: 'divination',
        question: '近期合作如何',
        casting: divinationMocks.casting,
        casting_instant: expect.any(String),
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      }),
    )
    expect(send).toHaveBeenCalledWith('这件事应该怎么处理？')
    wrapper.unmount()
  })

})
