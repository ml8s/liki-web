<template>
  <nav class="app-tabbar">
    <router-link
      to="/"
      exact-active-class="on"
      @pointerenter="preloadNavigation"
      @focusin="preloadNavigation"
    >
      <span>{{ $t('nav.workspace') }}</span>
    </router-link>
    <router-link
      to="/birth-profiles"
      active-class="on"
      @pointerenter="preloadNavigation"
      @focusin="preloadNavigation"
    >
      <span>{{ $t('nav.profiles') }}</span>
    </router-link>
    <router-link
      to="/chat"
      active-class="on"
      @pointerenter="preloadNavigation"
      @focusin="preloadNavigation"
    >
      <span>{{ $t('nav.chat') }}</span>
    </router-link>
    <router-link
      to="/account"
      active-class="on"
      @pointerenter="preloadNavigation"
      @focusin="preloadNavigation"
    >
      <span>{{ $t('nav.mine') }}</span>
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { createIntentPreloader } from '@/utils/routes'

const router = useRouter()
const preloadNavigation = createIntentPreloader(router, record => !String(record.name ?? '').startsWith('admin-'))
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
  font-size: var(--font-size-body);
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
