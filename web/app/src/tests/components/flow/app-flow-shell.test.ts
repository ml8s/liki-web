import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AppFlowShell from '@/components/flow/AppFlowShell.vue'

type ShellProps = {
  personLabel?: string
  domainLabel: string
  restoreVisible?: boolean
}

function mountShell(props: ShellProps = { domainLabel: '命理' }) {
  return mount(AppFlowShell, {
    props,
    slots: {
      default: '<p data-testid="consultation">consultation document</p>',
      actions: '<button type="button" data-testid="dock-action">send</button>',
    },
    global: { stubs: { FlowContextMenu: true } },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('AppFlowShell', () => {
  it('renders compact context and both flow layers', () => {
    const wrapper = mountShell({ personLabel: '张三', domainLabel: '命理' })

    expect(wrapper.get('.flow-status__person').text()).toBe('张三')
    expect(wrapper.get('.flow-status__domain').text()).toBe('命理')
    expect(wrapper.get('[data-testid="consultation"]').text()).toContain('consultation')
    expect(wrapper.get('[data-testid="dock-action"]').text()).toBe('send')
    expect(wrapper.findAll('main')).toHaveLength(1)
    expect(wrapper.get('.flow-status__announcement').text()).toBe('咨询上下文：张三，命理。')
    expect(wrapper.get('.flow-main').attributes('tabindex')).toBe('-1')
    expect(wrapper.find('.flow-restore').exists()).toBe(false)
  })

  it('announces an empty person without exposing a visible duplicate control', () => {
    const wrapper = mountShell({ domainLabel: '问卦' })

    expect(wrapper.get('.flow-status__announcement').text()).toBe('咨询上下文：选择档案，问卦。')
    expect(wrapper.get('.flow-status__announcement').classes()).toContain('app-live-region--screen-only')
  })

  it('shows restore only when requested and emits it', async () => {
    const hidden = mountShell({ domainLabel: '命理', restoreVisible: false })
    expect(hidden.find('.flow-restore').exists()).toBe(false)

    const wrapper = mountShell({ domainLabel: '命理', restoreVisible: true })
    await wrapper.get('.flow-restore').trigger('click')
    expect(wrapper.emitted('restore')).toHaveLength(1)
  })

  it('exposes a stable skip target and one main region', () => {
    const wrapper = mountShell()
    const skipLink = wrapper.get('.flow-skip-link')
    const main = wrapper.get('.flow-main')
    const mainId = main.attributes('id')

    expect(mainId).toBeTruthy()
    expect(skipLink.attributes('href')).toBe(`#${mainId}`)
    expect(wrapper.findAll('main')).toHaveLength(1)
  })

  it('keeps follow behavior outside the frozen shell contract', () => {
    const wrapper = mountShell({ domainLabel: '命理', restoreVisible: true })
    const options = wrapper.getComponent(AppFlowShell).vm.$options

    expect(wrapper.find('.flow-follow').exists()).toBe(false)
    expect(Object.keys(options.props ?? {})).toEqual([
      'personLabel',
      'domainLabel',
      'domains',
      'selectedDomainId',
      'partnerVisible',
      'restoreVisible',
      'sidecarOpen',
      'sidecarMode',
    ])
    expect(options.emits).toEqual(['restore', 'domain-select', 'context-select'])
    expect(Object.keys(wrapper.emitted())).toEqual([])
  })

  it('exposes the primary scroll element and focuses it for follow actions', async () => {
    const wrapper = mountShell()
    const main = wrapper.get('.flow-main').element as HTMLElement
    const focus = vi.spyOn(main, 'focus').mockImplementation(() => {})

    expect(wrapper.vm.getMainElement()).toBe(main)
    await wrapper.vm.scrollToBottom({ focusMain: true })

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('reduces scrolling motion when the user requests reduced motion', async () => {
    const scrollTo = vi.fn()
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: true }))
    const wrapper = mountShell()
    const main = wrapper.get('.flow-main').element
    Object.defineProperties(main, {
      scrollHeight: { configurable: true, value: 1_000 },
      clientHeight: { configurable: true, value: 400 },
      scrollTo: { configurable: true, value: scrollTo },
    })

    await wrapper.vm.scrollToBottom()

    expect(scrollTo).toHaveBeenCalledWith({
      top: 1_000,
      behavior: 'auto',
    })
  })
})
