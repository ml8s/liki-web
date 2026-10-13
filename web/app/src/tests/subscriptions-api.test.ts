import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchSubscriptionPayments, fetchSubscriptions, setSubscriptionCancellation } from '@/api/orders'

describe('subscriptions API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('lists subscriptions', async () => {
    const subscription = { id: 'sub-local', status: 'active' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { subscriptions: [subscription] } }),
      { status: 200 },
    )))
    await expect(fetchSubscriptions()).resolves.toEqual([subscription])
  })

  it('lists subscription payments with refund state', async () => {
    const payment = { id: 'payment-1', amount_minor_units: 1990, refunded_minor_units: 500, status: 'paid' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { payments: [payment] } }),
      { status: 200 },
    )))
    await expect(fetchSubscriptionPayments()).resolves.toEqual([payment])
  })

  it('updates cancellation with POST and DELETE', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { cancel_at_period_end: true } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { cancel_at_period_end: false } }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(setSubscriptionCancellation('sub-local', true)).resolves.toBe(true)
    await expect(setSubscriptionCancellation('sub-local', false)).resolves.toBe(false)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/me/subscriptions/sub-local/cancellation')
    expect(fetchMock.mock.calls[0]?.[1]?.method).toBe('POST')
    expect(fetchMock.mock.calls[1]?.[1]?.method).toBe('DELETE')
  })
})
