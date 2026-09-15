<template>
  <div class="app-shell">
    <AppSidebar :email="auth.user?.email ?? ''" @logout="logout" />

    <header class="app-topbar">
      <div class="brand font-serif">Liki<em>·</em>命理</div>
      <el-button size="small" @click="logout">退出</el-button>
    </header>

    <main class="app-main">
      <router-view />
    </main>

    <AppTabbar />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppTabbar from '@/components/app/AppTabbar.vue'

const auth = useAuthStore()
const router = useRouter()

onMounted(async () => {
  await auth.loadMe()
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

.app-topbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; background: var(--liki-ink); color: #fff;
  position: sticky; top: 0; z-index: 40;
}
.app-topbar .brand { font-size: 16px; font-weight: 700; letter-spacing: 2px; }
.app-topbar .brand em { color: var(--liki-gold); font-style: normal; }

.app-main { padding: 24px 16px 88px; }

@media (min-width: 1024px) {
  .app-topbar { display: none; }
  .app-main { margin-left: 240px; padding: 32px 40px 48px; max-width: 1200px; }
}
</style>