import { get, post } from './http'

export interface ChatProduct {
  id: string
  name: string
  description: string
  skills: string[]
  prices: Record<string, number>
  display_currency: string
  display_amount_minor: number
  purchased: boolean
}

export interface CheckoutResult {
  order_id: string
  payment_intent_id: string
  provider: 'dodo' | 'xunhu'
  status: 'created' | 'checkout_created'
  checkout_url: string
  status_token: string
}

export async function fetchChatProducts(): Promise<ChatProduct[]> {
  const payload = await get<{ products: ChatProduct[] }>('/api/chat/products')
  return payload.products ?? []
}

export function checkoutChatProduct(
  productId: string,
  locale = typeof navigator === 'undefined' ? 'zh-Hant' : navigator.language,
): Promise<CheckoutResult> {
  return post<CheckoutResult>(`/api/chat/products/${productId}/checkout`, { locale })
}
