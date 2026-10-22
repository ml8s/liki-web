<template>
  <aside class="app-sidebar">
    <router-link
      to="/"
      class="side-brand font-serif"
    >
      Liki<em>·</em>{{ $t('auth.brandTail') }}
    </router-link>
    <div class="side-slogan font-serif">
      {{ $t('auth.slogan') }}
    </div>

    <nav class="side-nav">
      <div class="nav-group">
        <router-link
          to="/"
          exact-active-class="on"
          :aria-current="isCurrent('/') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.workspace') }}
        </router-link>
        <router-link
          to="/birth-profiles"
          active-class="on"
          :aria-current="isCurrent('/birth-profiles') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.profiles') }}
        </router-link>
        <router-link
          to="/chat"
          active-class="on"
          :aria-current="isCurrent('/chat') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.chat') }}
        </router-link>
      </div>
      <div class="nav-group nav-group-secondary">
        <router-link
          to="/orders"
          active-class="on"
          :aria-current="isCurrent('/orders') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.orders') }}
        </router-link>
        <router-link
          to="/account"
          active-class="on"
          :aria-current="isCurrent('/account') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.account') }}
        </router-link>
      </div>
      <div
        v-if="auth.hasPartnerRole"
        class="nav-group nav-group-secondary"
      >
        <router-link
          to="/partners"
          exact-active-class="on"
          :aria-current="isCurrent('/partners') ? 'page' : undefined"
          @pointerenter="preloadNavigation"
          @focusin="preloadNavigation"
        >
          {{ $t('nav.partner') }}
        </router-link>
      </div>
    </nav>

    <div class="side-user">
      <span class="email">{{ email }}</span>
      <el-button
        size="small"
        plain
        @click="$emit('logout')"
      >
        {{ $t('nav.logout') }}
      </el-button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { createIntentPreloader } from '@/utils/routes'
import { useAuthStore } from '@/stores/auth'

defineProps<{ email: string }>()
defineEmits<{ logout: [] }>()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const preloadNavigation = createIntentPreloader(router, record => !String(record.name ?? '').startsWith('admin-'))

// 侧栏链接都是扁平路由，active 即 exact-active（/partners 显式用 exact-active-class）；
// 显式输出与 .on 同源的 aria-current="page"，不依赖 RouterLink 的内部判断。
function isCurrent(target: string) {
  return route.path === target
}
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  inset: 0 auto 0 0;
  width: 240px;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  background: var(--liki-ink);
  color: var(--liki-on-ink);
  padding: 24px 16px;
  z-index: var(--z-surface);
}
.side-brand { font-size: var(--font-size-title-md); font-weight: 700; letter-spacing: 2px; color: var(--liki-on-ink); text-decoration: none; }
.side-brand em { color: var(--liki-primary); font-style: normal; }
.side-slogan { font-size: var(--font-size-caption); color: var(--liki-on-ink-muted); margin: 6px 0 24px; }
.side-nav { display: flex; flex-direction: column; gap: 20px; flex: 1; }
.nav-group { display: flex; flex-direction: column; gap: 4px; }
.nav-group-secondary { border-top: 1px solid var(--liki-on-ink-line); padding-top: 16px; }
.side-nav a {
  position: relative;
  color: var(--liki-on-ink-link); text-decoration: none; padding: 10px 12px;
  border-radius: var(--radius-md); font-size: var(--font-size-body-lg);
  transition: background var(--liki-dur) var(--liki-ease), color var(--liki-dur) var(--liki-ease);
}
.side-nav a:hover { background: var(--liki-on-ink-hover); color: var(--liki-on-ink); }
.side-nav a.on { background: var(--liki-selected-overlay); color: var(--liki-on-ink); }
.side-nav a.on::before {
  content: '';
  position: absolute;
  left: 0; top: 8px; bottom: 8px;
  width: 2px;
  border-radius: 2px;
  background: var(--liki-primary);
}
.side-user { border-top: 1px solid var(--liki-on-ink-line); padding-top: 16px; display: flex; flex-direction: column; gap: 10px; }
.email { font-size: var(--font-size-label); opacity: .85; word-break: break-all; }
@media (max-width: 1023.98px) { .app-sidebar { display: none; } }
</style>
