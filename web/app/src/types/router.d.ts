import 'vue-router'
import type { AdminCapability } from '@/api/admin'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    adminCapability?: AdminCapability
    public?: boolean
    /** 该视图独占视口（自身负责滚动，例如会话页）：shell 会被锁到视口高度。 */
    viewportBound?: boolean
    title?: string
    titleKey?: string
  }
}
