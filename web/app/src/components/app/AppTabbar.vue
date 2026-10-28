<template>
  <nav class="app-tabbar">
    <router-link
      v-for="tab in tabs"
      :key="tab.label"
      :to="tab.to"
      :class="{ on: tab.active }"
      :aria-current="tab.active ? 'page' : undefined"
      @pointerenter="preloadNavigation"
      @focusin="preloadNavigation"
    >
      <span>{{ tab.label }}</span>
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { createIntentPreloader } from '@/utils/routes'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const preloadNavigation = createIntentPreloader(router, record => !String(record.name ?? '').startsWith('admin-'))

// 与桌面侧栏同一目的地模型：Workspace / Profiles / Chat / Orders-Account。
// 合作入口为定向邀请制：仅对已有合作角色的账号显示（数据来自 /api/me）。
const auth = useAuthStore()
const tabs = computed(() => {
  const items = [
    { to: { name: 'dashboard' }, label: t('nav.workspace'), active: route.name === 'dashboard' },
    { to: { name: 'birth-profiles' }, label: t('nav.profiles'), active: ['birth-profiles', 'birth-profile-new', 'birth-profile-detail'].includes(String(route.name)) },
    { to: { name: 'chat' }, label: t('nav.chat'), active: ['chat', 'chat-session'].includes(String(route.name)) },
    { to: { name: 'account' }, label: t('nav.me'), active: ['account', 'orders'].includes(String(route.name)) },
  ]
  if (auth.hasPartnerRole) {
    items.push({ to: { name: 'partner-center' }, label: t('nav.partner'), active: ['partner-center', 'partner-kol', 'partner-scout'].includes(String(route.name)) })
  }
  return items
})
</script>

<style scoped>
.app-tabbar {
  position: fixed;
  left: 0; right: 0; bottom: 0;
  display: flex;
  background: var(--liki-surface);
  border-top: 1px solid var(--liki-line);
  padding-bottom: env(safe-area-inset-bottom);
  z-index: var(--z-surface);
}
.app-tabbar a {
  position: relative;
  flex: 1;
  display: flex; align-items: center; justify-content: center;
  height: 54px;
  color: var(--liki-text-2);
  text-decoration: none;
  font-size: var(--font-size-label);
  transition: color var(--liki-dur) var(--liki-ease);
}
.app-tabbar a.on { color: var(--liki-primary); font-weight: 600; }
.app-tabbar a.on::before {
  content: '';
  position: absolute;
  top: 0; left: 28%; right: 28%;
  height: 2px;
  border-radius: 2px;
  background: var(--liki-primary);
}
@media (min-width: 1024px) { .app-tabbar { display: none; } }
</style>
