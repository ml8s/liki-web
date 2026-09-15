import { get } from './http'

export interface ChatOrder {
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

export async function fetchChatOrders(): Promise<ChatOrder[]> {
  const payload = await get<{ orders: ChatOrder[] }>('/api/chat/orders')
  return payload.orders ?? []
}
