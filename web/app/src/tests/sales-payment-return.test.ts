import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSalesStore } from '@/stores/sales-store'
import type { Entitlement, EntitlementSummary } from '@/api/me'
import { i18n } from '@/i18n'

// 支付回调是「浏览器先回来、webhook 后到」的时序：状态必须以后端确认的权益为准，
// 轮询预算耗尽时只能报「仍在开通」，不能报已开通。
const entitlement: Entitlement = {
  id: 'ent_1',
  user_id: 'usr_pay',
  product_id: 'liki-chat',
  skills: ['liki'],
  status: 'active',
  starts_at: '2026-10-01T00:00:00Z',
  expires_at: '2026-11-01T00:00:00Z',
}

async function stubWorkspace(entitledAfter: number) {
  const [me, chat, sales] = await Promise.all([
    import('@/api/me'),
    import('@/api/chat'),
    import('@/api/sales'),
  ])
  let refreshes = 0
  vi.spyOn(me, 'fetchEntitlements').mockImplementation(async (): Promise<EntitlementSummary> => {
    refreshes += 1
    const active = refreshes >= entitledAfter
    return {
      entitlements: active ? [entitlement] : [],
      allowed_skills: active ? ['liki'] : [],
    }
  })
  vi.spyOn(chat, 'fetchChatSessions').mockResolvedValue([])
  vi.spyOn(sales, 'fetchProducts').mockResolvedValue([])
  return () => refreshes
}

beforeEach(() => {
  setActivePinia(createPinia())
  sessionStorage.clear()
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('sales handlePaymentReturn', () => {
  it('does nothing without an order reference', async () => {
    const result = await useSalesStore().handlePaymentReturn({ status: 'paid' })
    expect(result).toEqual({ action: 'none' })
  })

  it('reports an unfinished payment', async () => {
    const sales = useSalesStore()
    const result = await sales.handlePaymentReturn({ order_id: 'order_1', status: 'unpaid' })
    expect(result).toEqual({ action: 'show_message', message: i18n.global.t('commerce.payment.pending'), state: 'unpaid' })
    expect(sales.paymentState).toBe('unpaid')
  })

  it('enters the chat once the entitlement lands during polling', async () => {
    vi.useFakeTimers()
    const refreshes = await stubWorkspace(2)
    const sales = useSalesStore()
    const pending = sales.handlePaymentReturn({ order_id: 'order_1', status: 'paid' })
    await vi.advanceTimersByTimeAsync(1_000)
    await expect(pending).resolves.toEqual({ action: 'enter_chat', scenarioId: 'natal' })
    expect(refreshes()).toBe(2)
    expect(sales.paymentState).toBe('confirmed')
    expect(sales.paymentMessage).toBe(i18n.global.t('commerce.payment.active'))
  })

  it('stops after the polling budget and reports that fulfillment is still running', async () => {
    vi.useFakeTimers()
    const refreshes = await stubWorkspace(Number.POSITIVE_INFINITY)
    const sales = useSalesStore()
    const pending = sales.handlePaymentReturn({ order_id: 'order_1', status: 'paid' })
    await vi.advanceTimersByTimeAsync(60_000)
    await expect(pending).resolves.toEqual({
      action: 'show_message',
      message: i18n.global.t('commerce.payment.processing'),
      state: 'verifying',
    })
    // 12 次指数退避（1s→5s）：预算固定，不能无限轮询。
    expect(refreshes()).toBe(12)
    expect(sales.paymentMessage).not.toBe(i18n.global.t('commerce.payment.active'))
  })
})


  it('preserves the scenario selected before checkout', async () => {
    vi.useFakeTimers()
    await stubWorkspace(1)
    sessionStorage.setItem('liki.pending-payment-scenario', 'divination')
    const sales = useSalesStore()
    const pending = sales.handlePaymentReturn({ order_id: 'order_1', status: 'paid' })
    await vi.advanceTimersByTimeAsync(1)
    await expect(pending).resolves.toEqual({ action: 'enter_chat', scenarioId: 'divination' })
  })

describe('sales subscription return', () => {
  it('waits for the entitlement after a subscription checkout', async () => {
    vi.useFakeTimers()
    const refreshes = await stubWorkspace(1)
    const sales = useSalesStore()
    const pending = sales.handlePaymentReturn({ subscription_id: 'sub_1' })
    await vi.advanceTimersByTimeAsync(1)
    await expect(pending).resolves.toEqual({ action: 'enter_chat', scenarioId: 'natal' })
    expect(refreshes()).toBe(1)
    expect(sales.paymentState).toBe('confirmed')
  })

  it('reports a cancelled subscription checkout without polling', async () => {
    const sales = useSalesStore()
    const result = await sales.handlePaymentReturn({ subscription_state: 'cancelled' })
    expect(result).toEqual({
      action: 'show_message',
      message: i18n.global.t('commerce.payment.cancelled'),
      state: 'unpaid',
    })
    expect(sales.paymentState).toBe('unpaid')
  })
})
