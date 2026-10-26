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

async function createTestRouter(initialUrl = '/flow') {
  const router = createRouter({
    history: createMemoryHistory('/app/'),
    routes: [
      { path: '/consultation', name: 'app-consultation', component: { template: '<div />' } },
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
vi.mock('@/composables/useToolRecords', () => ({
  useToolRecords: () => ({ save: saveToolRecord }),
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

    const router = await createTestRouter('/flow')
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

    const router = await createTestRouter('/flow?seed=huangli')
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

    const router = await createTestRouter('/flow?domain=divination')
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
    const router = await createTestRouter('/flow?domain=divination')
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
