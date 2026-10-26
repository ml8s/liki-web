import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AppStatusAlert from '@/components/app/AppStatusAlert.vue'

describe('AppStatusAlert', () => {
  it('announces errors assertively and exposes a recovery action', async () => {
    const wrapper = mount(AppStatusAlert, {
      props: { title: '加载产品失败', type: 'error' },
      slots: { default: '<button type="button">重试</button>' },
    })

    expect(wrapper.get('[role="alert"]').text()).toContain('加载产品失败')
    expect(wrapper.get('button').text()).toBe('重试')
    wrapper.unmount()
  })

  it('announces informational states without interrupting assistive tech', () => {
    const wrapper = mount(AppStatusAlert, {
      props: { title: '支付成功', type: 'success' },
    })

    expect(wrapper.get('[role="status"]').text()).toContain('支付成功')
    expect(wrapper.find('button').exists()).toBe(false)
    wrapper.unmount()
  })
})
