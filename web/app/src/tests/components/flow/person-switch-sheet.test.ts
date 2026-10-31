import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import PersonSwitchSheet from '@/components/flow/PersonSwitchSheet.vue'
import type { BirthProfile } from '@/api/birth-profiles'

function birthProfile(id: string, nickname: string): BirthProfile {
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

type PersonSwitchSheetProps = {
  modelValue: boolean
  profiles: BirthProfile[]
  selectedPersonId: string | null
  loading?: boolean
  error?: string
}

async function openPersonSheet(props: PersonSwitchSheetProps) {
  const wrapper = mount(PersonSwitchSheet, {
    props,
    attachTo: document.body,
  })
  await nextTick()
  return wrapper
}

async function click(selector: string) {
  const element = document.querySelector<HTMLElement>(selector)
  expect(element, selector).toBeInstanceOf(HTMLElement)
  element!.click()
  await nextTick()
}

describe('PersonSwitchSheet', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('uses a native radio group for one-person selection', async () => {
    const profiles = [birthProfile('person-1', '张三'), birthProfile('person-2', '李四')]
    const wrapper = await openPersonSheet({
      modelValue: true,
      profiles,
      selectedPersonId: 'person-1',
    })

    const group = document.querySelector('[role="radiogroup"], fieldset')
    expect(group).not.toBeNull()
    const inputs = [...document.querySelectorAll<HTMLInputElement>('.person-option input[type="radio"]')]
    expect(inputs).toHaveLength(2)
    expect(inputs[0]?.checked).toBe(true)
    expect(inputs[1]?.checked).toBe(false)
    expect(document.querySelectorAll('.person-option--selected')).toHaveLength(1)
    expect(document.querySelector('.person-check')?.textContent).toContain('✓')

    inputs[1]!.click()
    await nextTick()
    expect(wrapper.emitted('select')).toEqual([['person-2']])
    wrapper.unmount()
  })

  it('uses the shared profile summary rather than a bare date', async () => {
    const wrapper = await openPersonSheet({
      modelValue: true,
      profiles: [birthProfile('person-1', '张三')],
      selectedPersonId: 'person-1',
    })

    const summary = document.querySelector('.person-summary')
    expect(summary?.textContent).toContain('2000-01-01')
    wrapper.unmount()
  })

  it('emits create, manage, and retry actions without routing', async () => {
    const wrapper = await openPersonSheet({
      modelValue: true,
      profiles: [birthProfile('person-1', '张三')],
      selectedPersonId: null,
    })

    await click('.person-footer .person-action:not(.person-action--secondary)')
    expect(wrapper.emitted('create')).toHaveLength(1)

    await click('.person-action--secondary')
    expect(wrapper.emitted('manage')).toHaveLength(1)
    expect(wrapper.emitted()).not.toHaveProperty('select')
    wrapper.unmount()
  })

  it('presents loading and error states and supports recovery', async () => {
    const loading = await openPersonSheet({
      modelValue: true,
      profiles: [],
      selectedPersonId: null,
      loading: true,
    })
    expect(document.querySelector('[role="status"] [aria-busy="true"]')).not.toBeNull()
    expect(document.querySelector('.person-option')).toBeNull()
    loading.unmount()
    document.body.innerHTML = ''

    const failed = await openPersonSheet({
      modelValue: true,
      profiles: [],
      selectedPersonId: null,
      error: '无法加载档案',
    })
    const alert = document.querySelector('[role="alert"]')
    expect(alert?.textContent).toContain('无法加载档案')
    await click('.person-state .person-action')
    expect(failed.emitted('retry')).toHaveLength(1)
    failed.unmount()
  })

  it('shows an empty state while keeping profile actions available', async () => {
    const wrapper = await openPersonSheet({
      modelValue: true,
      profiles: [],
      selectedPersonId: null,
    })

    expect(document.querySelector('[role="status"].person-empty')?.textContent).toContain('还没有档案')
    expect(document.querySelector('.person-option')).toBeNull()
    await click('.person-footer .person-action:not(.person-action--secondary)')
    expect(wrapper.emitted('create')).toHaveLength(1)
    wrapper.unmount()
  })
})
