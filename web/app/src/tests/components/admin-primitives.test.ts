import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'

describe('admin primitives', () => {
  it('maps one status to one operator label and semantic color', () => {
    const requested = mount(AdminStatusTag, { props: { status: 'requested' } })
    expect(requested.text()).toBe('待审核')
    expect(requested.find('.el-tag').classes()).toContain('el-tag--warning')

    const unknown = mount(AdminStatusTag, { props: { status: 'future.state' } })
    expect(unknown.text()).toBe('future.state')
    expect(unknown.find('.el-tag').classes()).toContain('el-tag--info')
  })

  it('keeps page copy and actions in one heading contract', () => {
    const wrapper = mount(AdminPageHeader, {
      props: { title: '合作伙伴审计', description: '检查 partner 域操作。' },
      slots: { actions: '<button type="button">刷新</button>' },
    })
    expect(wrapper.find('h1').text()).toBe('合作伙伴审计')
    expect(wrapper.find('p').text()).toBe('检查 partner 域操作。')
    expect(wrapper.find('button').text()).toBe('刷新')
  })
})

describe('AdminRecordCard', () => {
  it('renders formatted fields and scoped actions', async () => {
    const wrapper = mount(AdminRecordCard, {
      props: {
        fields: [
          { key: 'amount', label: '金额', value: 12300, kind: 'money', currency: 'USD' },
          { key: 'status', label: '状态', value: 'requested', kind: 'status' },
          { key: 'reference', label: '关联对象', value: 'pi_123', kind: 'mono' },
        ],
      },
      slots: { actions: '<button type="button">批准</button>' },
    })
    expect(wrapper.text()).toContain('USD 123.00')
    expect(wrapper.text()).toContain('待审核')
    expect(wrapper.text()).toContain('pi_123')
    expect(wrapper.text()).toContain('批准')
  })

  it('lets a caller override a field while preserving card layout', () => {
    const wrapper = mount(AdminRecordCard, {
      props: {
        fields: [
          { key: 'capabilities', label: '能力', value: 'partner.read' },
          { key: 'other', label: '其他', value: 'value' },
        ],
      },
      slots: {
        field: `<template #field="{ field }">
          <strong v-if="field.key === 'capabilities'">partner.read</strong>
          <span v-else>{{ field.value }}</span>
        </template>`,
      },
    })
    expect(wrapper.find('strong').text()).toBe('partner.read')
    expect(wrapper.text()).toContain('value')
  })
})
