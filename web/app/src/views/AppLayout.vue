<template>
  <el-container class="app-shell">
    <el-header height="64px" class="app-header">
      <div class="brand">Liki App</div>
      <nav class="nav">
        <router-link to="/">工作台</router-link>
        <router-link to="/reports">报告</router-link>
        <router-link to="/orders">订单</router-link>
        <router-link to="/account">账户</router-link>
      </nav>
      <div class="actions">
        <span v-if="auth.user" class="email">{{ auth.user.email }}</span>
        <el-button size="small" @click="logout">退出</el-button>
      </div>
    </el-header>
    <el-main class="app-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

onMounted(async () => {
  await auth.loadMe()
})

async function logout() {
  await auth.logout()
  await router.push({ name: 'login' })
}
</script>

<style scoped>
.app-shell { min-height: 100vh; background: #f7f7f5; }
.app-header { display: flex; align-items: center; gap: 24px; background: #191919; color: #fff; }
.brand { font-weight: 700; font-size: 18px; }
.nav { display: flex; gap: 18px; flex: 1; }
.nav a { color: #d9d9d9; text-decoration: none; }
.nav a.router-link-active { color: #fff; }
.actions { display: flex; align-items: center; gap: 12px; }
.email { font-size: 13px; opacity: .8; }
.app-main { max-width: 1200px; margin: 0 auto; width: 100%; }
</style>
