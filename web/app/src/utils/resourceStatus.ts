import { ApiError } from '@/api/http'

export type ResourceFailureKind =
  | 'offline'
  | 'unreachable'
  | 'permission'
  | 'server'
  | 'unknown'

export function classifyResourceFailure(error: unknown): Exclude<ResourceFailureKind, 'offline'> {
  if (error instanceof ApiError) {
    if (error.status === 0) return 'unreachable'
    if (error.status === 401 || error.status === 403) return 'permission'
    if (error.status >= 500) return 'server'
    return 'unknown'
  }
  if (error instanceof TypeError || error instanceof DOMException) return 'unreachable'
  return 'unknown'
}
