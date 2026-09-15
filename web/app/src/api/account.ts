import { post } from './http'

export interface ErasureResult {
  status: string
  profiles_cleared: number
  messages_cleared: number
}

export function erasureAccount(): Promise<ErasureResult> {
  return post<ErasureResult>('/api/account/erasure')
}