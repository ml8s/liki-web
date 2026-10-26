import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import EngineResultBoundary from '@shared/components/engine/EngineResultBoundary.vue'

describe('EngineResultBoundary', () => {
  it('renders a loading state while there is no result', () => {
    const wrapper = mount(EngineResultBoundary, {
      props: { status: 'loading' },
    })

    expect(wrapper.find('[role="status"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('加载中…')
    expect(wrapper.attributes('aria-busy')).toBe('true')
  })

  it('renders a typed result through the default slot', () => {
    const wrapper = mount(EngineResultBoundary, {
      props: {
        status: 'success',
        data: { chart: 'abc' },
      },
      slots: {
        default: `<template #default="{ result }">
          <div data-testid="result">{{ result.chart }}</div>
        </template>`,
      },
    })

    expect(wrapper.get('[data-testid="result"]').text()).toBe('abc')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('renders an error and emits retry when no result exists', async () => {
    const wrapper = mount(EngineResultBoundary, {
      props: {
        status: 'error',
        error: { name: 'EngineResultError', message: 'engine unavailable' },
      },
    })

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('engine unavailable')

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('allows stale data and an error notice to coexist', () => {
    const wrapper = mount(EngineResultBoundary, {
      props: {
        status: 'success',
        data: { chart: 'stale' },
        error: { name: 'EngineResultError', message: 'refresh failed' },
      },
      slots: {
        default: `<template #default="{ result }">
          <div data-testid="stale-result">{{ result.chart }}</div>
        </template>`,
        error: `<template #error="{ error }">
          <div data-testid="stale-error">{{ error.message }}</div>
        </template>`,
      },
    })

    expect(wrapper.get('[data-testid="stale-result"]').text()).toBe('stale')
    expect(wrapper.get('[data-testid="stale-error"]').text()).toBe('refresh failed')
  })

  it('renders an explicit empty state', () => {
    const wrapper = mount(EngineResultBoundary, {
      props: { status: 'idle' },
      slots: { empty: '<div data-testid="empty">no input</div>' },
    })

    expect(wrapper.get('[data-testid="empty"]').text()).toBe('no input')
  })
})
