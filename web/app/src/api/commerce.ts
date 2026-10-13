import { get, post } from './http'

export interface ChatProduct {
  id: string
  name: string
  description: string
  skills: string[]
  display_currency: string
  display_amount_minor: number
  purchase_options: PurchaseOption[]
  purchased: boolean
}

export interface MoneyMinor {
  currency_code: 'CNY' | 'USD'
  minor_units: number
}

export interface PurchaseOption {
  billing_mode: 'one_time' | 'subscription'
  price: MoneyMinor
}

export interface OneTimeCheckoutResult {
  order_id: string
  payment_intent_id: string
  provider: 'dodo' | 'xunhu'
  status: 'created' | 'checkout_created'
  checkout_url: string
  status_token: string
}

export interface SubscriptionCheckoutResult {
  subscription_id: string
  provider: 'dodo'
  status: 'pending'
  checkout_url: string
}

export type CheckoutResult = OneTimeCheckoutResult | SubscriptionCheckoutResult

export async function fetchChatProducts(): Promise<ChatProduct[]> {
  const payload = await get<{ products: ChatProduct[] }>('/api/chat/products')
  return payload.products ?? []
}

export function checkoutChatProduct(
  productId: string,
  billingMode: 'one_time' | 'subscription',
  locale = typeof navigator === 'undefined' ? 'zh-Hant' : navigator.language,
): Promise<CheckoutResult> {
  return post<CheckoutResult>(`/api/chat/products/${productId}/checkout`, { billing_mode: billingMode, locale })
}
