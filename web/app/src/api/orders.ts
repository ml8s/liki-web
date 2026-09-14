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

export function fetchChatOrders(): Promise<ChatOrder[]> {
  return get<ChatOrder[]>('/api/chat/orders')
}
