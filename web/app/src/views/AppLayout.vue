<template>
  <div class="app-shell">
    <a href="#app-main" class="skip-link">跳到主要内容</a>
    <AppSidebar :email="auth.user?.email ?? ''" @logout="logout" />

    <header class="app-topbar">
      <div class="brand font-serif">Liki<em>·</em>命理</div>
      <el-button size="small" @click="logout">退出</el-button>
    </header>

    <main id="app-main" ref="mainEl" class="app-main" tabindex="-1">
      <router-view />
    </main>

    <AppTabbar />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppTabbar from '@/components/app/AppTabbar.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const mainEl = ref<HTMLElement | null>(null)

onMounted(async () => {
  try {
    await auth.loadMe()
  } catch {
    // 路由守卫已处理 401 重定向；网络异常不崩溃整页。
  }
})

// 路由切换后把焦点移到主内容，供键盘/读屏用户感知页面已变更。
watch(() => route.fullPath, async () => {
  await nextTick()
  mainEl.value?.focus()
})

async function logout() {
  await auth.logout()
  useEntitlementsStore().reset()
  useChatStore().reset()
  await router.push({ name: 'login' })
}
</script>

<style scoped>
.app-shell { min-height: 100vh; background: var(--liki-paper); }

.skip-link {
  position: absolute; left: -9999px; top: 0; z-index: 100;
  background: var(--liki-primary); color: #fff;
  padding: 10px 16px; border-radius: 0 0 8px 0; text-decoration: none;
}
.skip-link:focus { left: 0; }

.app-topbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; background: var(--liki-ink); color: #fff;
  position: sticky; top: 0; z-index: 40;
}
.app-topbar .brand { font-size: 16px; font-weight: 700; letter-spacing: 2px; }
.app-topbar .brand em { color: var(--liki-gold); font-style: normal; }

.app-main { padding: 24px 16px 88px; }
.app-main:focus { outline: none; }

@media (min-width: 1024px) {
  .app-topbar { display: none; }
  .app-main { margin-left: 240px; padding: 32px 40px 48px; max-width: 1200px; }
}
</style>
