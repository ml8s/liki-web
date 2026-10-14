import 'vue-router'
import type { AdminCapability } from '@/api/admin'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    adminCapability?: AdminCapability
    public?: boolean
    title?: string
    titleKey?: string
  }
}
