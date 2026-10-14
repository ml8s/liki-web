import { del, get, post } from './http'

export interface Order {
  id: string
  user_id: string
  email: string
  product_id: string
  amount_minor: number
  currency: string
  status: 'pending' | 'paid' | 'canceled'
  payment_intent_id: string
  created_at: string
  updated_at: string
  paid_at: string | null
}

export async function fetchOrders(): Promise<Order[]> {
  const payload = await get<{ orders: Order[] }>('/api/sales/orders')
  return payload.orders ?? []
}

export interface Subscription {
  id: string
  user_id: string
  email: string
  product_id: string
  billing_mode: 'subscription'
  provider: 'dodo'
  provider_subscription_id: string
  status: 'pending' | 'active' | 'on_hold' | 'cancelled' | 'expired' | 'failed'
  currency_code: 'CNY' | 'USD'
  current_period_start: string | null
  current_period_end: string | null
  next_billing_at: string | null
  cancel_at_period_end: boolean
  created_at: string
  updated_at: string
}

export interface SubscriptionPayment {
  id: string
  subscription_id: string
  product_id: string
  provider: 'dodo'
  provider_payment_id: string
  amount_minor_units: number
  refunded_minor_units: number
  currency_code: 'CNY' | 'USD'
  status: 'paid' | 'refunded'
  period_start: string | null
  period_end: string | null
  created_at: string
  updated_at: string
}

export async function fetchSubscriptions(): Promise<Subscription[]> {
  const payload = await get<{ subscriptions: Subscription[] }>('/api/sales/subscriptions')
  return payload.subscriptions ?? []
}

export async function fetchSubscriptionPayments(): Promise<SubscriptionPayment[]> {
  const payload = await get<{ payments: SubscriptionPayment[] }>('/api/sales/subscription-payments')
  return payload.payments ?? []
}

export async function setSubscriptionCancellation(subscriptionId: string, cancel: boolean): Promise<boolean> {
  const payload = cancel
    ? await post<{ cancel_at_period_end: boolean }>(`/api/sales/subscriptions/${subscriptionId}/cancellation`)
    : await del<{ cancel_at_period_end: boolean }>(`/api/sales/subscriptions/${subscriptionId}/cancellation`)
  return payload.cancel_at_period_end
}
