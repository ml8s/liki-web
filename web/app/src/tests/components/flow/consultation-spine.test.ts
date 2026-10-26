import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ConsultationSpine from '@/components/flow/ConsultationSpine.vue'

const baseProps = {
  domainLabel: '命理',
}

describe('ConsultationSpine', () => {
  it('renders person and domain as the document context', () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, personLabel: '张三' },
    })

    expect(wrapper.find('h1').text()).toBe('张三')
    expect(wrapper.find('h1').attributes('id')).toBe(
      wrapper.find('section').attributes('aria-labelledby'),
    )
    expect(wrapper.find('.spine-domain').text()).toBe('命理')
    expect(wrapper.attributes('data-state')).toBe('empty')
  })

  it('uses profile-free copy when no person is selected', () => {
    const wrapper = mount(ConsultationSpine, { props: baseProps })

    expect(wrapper.find('h1').text()).toBe('命理')
    expect(wrapper.find('.app-empty__text').text()).toBe('还没有档案')
  })

  it('renders a polite loading state and hides content', () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, loading: true, hasProfile: true },
      slots: { default: ' consultation ' },
    })

    const status = wrapper.find('.spine-state')
    expect(wrapper.attributes('data-state')).toBe('loading')
    expect(status.attributes('role')).toBe('status')
    expect(status.attributes('aria-live')).toBe('polite')
    expect(status.attributes('aria-busy')).toBe('true')
    expect(status.find('.el-skeleton').exists()).toBe(true)
    expect(status.text()).toContain('加载中…')
    expect(wrapper.find('.spine-content').exists()).toBe(false)
  })

  it('renders an error politely and emits retry-profiles once', async () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, error: '档案加载失败', hasProfile: true },
    })

    expect(wrapper.attributes('data-state')).toBe('error')
    expect(wrapper.find('.spine-state').attributes('aria-live')).toBe('polite')
    expect(wrapper.find('.spine-error p').text()).toBe('档案加载失败')
    expect(wrapper.find('.spine-content').exists()).toBe(false)

    await wrapper.find('.spine-action').trigger('click')
    expect(wrapper.emitted('retry-profiles')).toHaveLength(1)
  })

  it('renders the empty state and emits create-profile once', async () => {
    const wrapper = mount(ConsultationSpine, { props: baseProps })

    expect(wrapper.attributes('data-state')).toBe('empty')
    expect(wrapper.find('.spine-state').attributes('role')).toBe('status')
    expect(wrapper.find('.app-empty__text').text()).toBe('还没有档案')

    await wrapper.find('.spine-action').trigger('click')
    expect(wrapper.emitted('create-profile')).toHaveLength(1)
  })

  it('renders profile-owned content in reading order without status content', () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, personLabel: '张三', hasProfile: true },
      slots: { default: '<section>命盘摘要</section>' },
    })

    expect(wrapper.attributes('data-state')).toBe('ready')
    expect(wrapper.find('.spine-state').text()).toBe('')
    const texts = wrapper.findAll('.spine-context, .spine-content').map(
      node => node.text(),
    )
    expect(texts).toEqual(['张三命理', '命盘摘要'])
  })

  it('keeps future content out of the transient live region', () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, hasProfile: true },
      slots: {
        default:
          '<section aria-labelledby="reading-title"><h2 id="reading-title">解读</h2></section>',
      },
    })

    expect(wrapper.find('.spine-state').text()).toBe('')
    expect(wrapper.find('#reading-title').text()).toBe('解读')
    expect(wrapper.find('h1').element.nextElementSibling?.textContent).toContain('命理')
  })

  it('gives loading precedence over error and profile state', () => {
    const wrapper = mount(ConsultationSpine, {
      props: {
        ...baseProps,
        loading: true,
        error: '档案加载失败',
        hasProfile: false,
      },
    })

    expect(wrapper.attributes('data-state')).toBe('loading')
    expect(wrapper.find('.spine-error').exists()).toBe(false)
    expect(wrapper.find('.app-empty__text').exists()).toBe(false)
  })

  it('gives error precedence over the empty profile state', () => {
    const wrapper = mount(ConsultationSpine, {
      props: { ...baseProps, error: '档案加载失败' },
    })

    expect(wrapper.attributes('data-state')).toBe('error')
    expect(wrapper.find('.spine-error').text()).toContain('档案加载失败')
    expect(wrapper.find('.app-empty__text').exists()).toBe(false)
  })

  it('allows long person and domain labels to wrap', () => {
    const wrapper = mount(ConsultationSpine, {
      props: {
        personLabel: '非常长的档案名称用于检查换行和最小宽度约束',
        domainLabel: '非常长的领域名称用于检查换行和最小宽度约束',
        hasProfile: true,
      },
    })

    expect(wrapper.find('.spine-title').text()).toContain('最小宽度约束')
    expect(wrapper.find('.spine-domain').text()).toContain('最小宽度约束')
  })
})
