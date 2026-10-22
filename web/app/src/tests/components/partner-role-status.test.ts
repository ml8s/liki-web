import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PartnerRoleStatus from '@/components/partner/PartnerRoleStatus.vue'

const messages = {
  'partner.frozenTitle': 'Frozen: {role}',
  'partner.frozenDescription': 'Account frozen',
  'partner.roleConflictTitle': 'Role conflict',
  'partner.roleConflictDescription': 'Other role assigned',
  'partner.pendingTitle': 'Pending: {role}',
  'partner.kol.pendingDescription': 'KOL pending',
  'partner.scout.pendingDescription': 'Scout pending',
  'partner.kol.applyTitle': 'Apply KOL',
  'partner.kol.applyDescription': 'KOL description',
  'partner.kol.noteLabel': 'Note',
  'partner.kol.notePlaceholder': 'Write note',
  'partner.submitApplication': 'Submit',
  'partner.rejectedTitle': 'Rejected: {reason}',
  'partner.rejectedTitleNoReason': 'Rejected',
}

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: messages } })

const stubs = {
  PartnerSettlement: { template: '<div data-testid="settlement" />' },
  ElResult: { template: '<div data-testid="result" />' },
  ElAlert: { template: '<div data-testid="alert" />' },
  ElCard: { template: '<div data-testid="card"><slot /></div>' },
  ElForm: { template: '<form @submit.prevent="$emit(\'submit\')"><slot /></form>' },
  ElFormItem: { template: '<div><slot /></div>' },
  ElInput: { template: '<textarea v-model="modelValue" />' },
  ElButton: { template: '<button type="submit"><slot /></button>' },
}

function mountStatus(props: Record<string, unknown>) {
  return mount(PartnerRoleStatus, {
    props: {
      role: 'kol',
      roleName: 'KOL',
      roleStatus: '',
      otherRoleAssigned: false,
      application: null,
      submitting: false,
      ...props,
    },
    global: { plugins: [i18n], stubs },
  })
}

describe('PartnerRoleStatus', () => {
  it('shows settlement when retired', () => {
    const wrapper = mountStatus({ roleStatus: 'retired' })
    expect(wrapper.find('[data-testid="settlement"]').exists()).toBe(true)
  })

  it('shows frozen result', () => {
    const wrapper = mountStatus({ roleStatus: 'frozen' })
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(true)
  })

  it('shows role conflict result', () => {
    const wrapper = mountStatus({ otherRoleAssigned: true })
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(true)
  })

  it('shows pending result', () => {
    const wrapper = mountStatus({ application: { status: 'pending', decision_reason: '' } })
    expect(wrapper.find('[data-testid="result"]').exists()).toBe(true)
  })

  it('shows apply card with form when no status', async () => {
    const wrapper = mountStatus({})
    expect(wrapper.find('[data-testid="card"]').exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('shows rejected alert when application was rejected', () => {
    const wrapper = mountStatus({
      application: { status: 'rejected', decision_reason: 'low quality' },
    })
    expect(wrapper.find('[data-testid="alert"]').exists()).toBe(true)
  })
})
