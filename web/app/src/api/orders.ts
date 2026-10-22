import { del, get, post } from './http'

export interface Order {
  id: string
  product_id: string
  amount_minor: number
  currency: string
  status: 'pending' | 'paid' | 'canceled'
  payment_intent_id: string
  /** 支付 intent 状态（BFF 注解）：created/checkout_created/paid/refunded/failed/expired。 */
  intent_status?: OrderIntentStatus
  created_at: string
  updated_at: string
  paid_at: string | null
}

export type OrderIntentStatus = 'created' | 'checkout_created' | 'paid' | 'refunded' | 'failed' | 'expired'

export async function fetchOrders(): Promise<Order[]> {
  const payload = await get<{ orders: Order[] }>('/api/sales/orders')
  return payload.orders ?? []
}

export function cancelOrder(orderId: string): Promise<{ status: string }> {
  return post<{ status: string }>(`/api/sales/orders/${orderId}/cancel`)
}

export interface OrderCheckoutResult {
  order_id: string
  payment_intent_id: string
  provider: string
  status: string
  checkout_url: string
  qr_code_url?: string
  status_token: string
}

/** 未付订单继续支付：同一订单开新支付会话并返回新的收银台链接。 */
export function resumeOrderCheckout(orderId: string): Promise<OrderCheckoutResult> {
  return post<OrderCheckoutResult>(`/api/sales/orders/${orderId}/checkout`)
}

export interface OrderPaymentInfo {
  provider: string
  status: OrderIntentStatus
  created_at: string
  updated_at: string
  paid_at: string | null
  refunded_at: string | null
  refunded_amount_minor: number
}

export interface OrderDetail {
  order: Order
  payment?: OrderPaymentInfo
}

export function fetchOrderDetail(orderId: string): Promise<OrderDetail> {
  return get<OrderDetail>(`/api/sales/orders/${orderId}`)
}

export interface Subscription {
  id: string
  product_id: string
  billing_mode: 'subscription'
  provider: 'dodo'
  status: 'pending' | 'active' | 'on_hold' | 'cancelled' | 'expired' | 'failed'
  currency_code: 'CNY' | 'USD'
  current_period_start: string | null
  current_period_end: string | null
  next_billing_at: string | null
  cancel_at_period_end: boolean
  cancelled_at?: string | null
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
