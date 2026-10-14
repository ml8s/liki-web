import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
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
