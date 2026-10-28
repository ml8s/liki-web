import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DomainChooserSheet from '@/components/flow/DomainChooserSheet.vue'
import PersonSwitchSheet from '@/components/flow/PersonSwitchSheet.vue'
import { appDomains } from '@shared/domain/flow'
import type { AppDomain, AppDomainId } from '@shared/domain/flow'
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

type DomainChooserSheetProps = {
  modelValue: boolean
  domains: readonly AppDomain[]
  selectedDomainId: AppDomainId
}

async function openPersonSheet(props: PersonSwitchSheetProps) {
  const wrapper = mount(PersonSwitchSheet, {
    props,
    attachTo: document.body,
  })
  await nextTick()
  return wrapper
}

async function openDomainSheet(props: DomainChooserSheetProps) {
  const wrapper = mount(DomainChooserSheet, {
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

describe('DomainChooserSheet', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('exposes exactly the four consultation domains and no legacy products', () => {
    expect(appDomains).toHaveLength(4)
    expect(appDomains.map(({ id }) => id)).toEqual([
      'natal',
      'divination',
      'fengshui',
      'naming',
    ])
  })

  it('uses a native radio group and shows a non-color selected cue', async () => {
    const wrapper = await openDomainSheet({
      modelValue: true,
      domains: appDomains,
      selectedDomainId: 'divination',
    })

    const inputs = [...document.querySelectorAll<HTMLInputElement>('.domain-option input[type="radio"]')]
    expect(inputs).toHaveLength(4)
    expect(inputs.find(input => input.value === 'divination')?.checked).toBe(true)
    expect(document.querySelectorAll('.domain-option--selected')).toHaveLength(1)
    expect(document.querySelector('.domain-check')?.textContent).toContain('✓')

    const naming = inputs.find(input => input.value === 'naming')
    naming!.click()
    await nextTick()
    expect(wrapper.emitted('select')).toEqual([['naming']])
    wrapper.unmount()
  })

  it('provides a visible close control and an accessible dialog name', async () => {
    const wrapper = await openDomainSheet({
      modelValue: true,
      domains: appDomains,
      selectedDomainId: 'natal',
    })

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog?.getAttribute('aria-label')).toBe('选择领域')
    const close = document.querySelector<HTMLButtonElement>('.flow-sheet__close')
    expect(close?.getAttribute('aria-label')).toBe('关闭')
    close!.click()
    await nextTick()
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])
    wrapper.unmount()
  })

  it('offers a new consultation action', async () => {
    const wrapper = await openDomainSheet({
      modelValue: true,
      domains: appDomains,
      selectedDomainId: 'natal',
    })

    await click('.new-consultation')
    expect(wrapper.emitted('new-consultation')).toHaveLength(1)
    wrapper.unmount()
  })

  it('keeps the modal focus trap and backdrop close affordance available', async () => {
    const wrapper = await openDomainSheet({
      modelValue: true,
      domains: appDomains,
      selectedDomainId: 'natal',
    })

    const overlay = document.querySelector<HTMLElement>('.el-overlay')
    expect(document.querySelector('[role="dialog"]')?.getAttribute('aria-modal')).toBe('true')
    expect(document.querySelector('.el-drawer__sr-focus')).not.toBeNull()
    overlay?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    overlay?.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
    overlay?.click()
    await nextTick()
    expect(document.querySelector('.el-drawer')?.classList.contains('open')).toBe(false)
    wrapper.unmount()
  })
})
