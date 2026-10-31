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
    /** 浏览器标签里的页面名（`<页面名> · <站点名>`）。业务页面用本地化键。 */
    titleKey?: string
    /** 同上，但用于只在中文后台出现的固定页名；声明了就覆盖 titleKey。 */
    title?: string
  }
}
