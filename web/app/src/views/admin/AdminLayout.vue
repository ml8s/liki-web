<template>
  <div class="admin-shell">
    <header class="admin-topbar">
      <div class="brand">
        Liki <span>Admin</span>
      </div>
      <nav
        v-if="admin.status === 'ready'"
        class="admin-nav"
        aria-label="管理导航"
      >
        <RouterLink
          to="/admin"
          exact-active-class="on"
        >
          总览
        </RouterLink>
        <RouterLink
          v-if="admin.can('entitlement.read')"
          to="/admin/entitlements"
          active-class="on"
        >
          权益
        </RouterLink>
        <RouterLink
          v-if="admin.can('partner.read')"
          to="/admin/partners"
          active-class="on"
        >
          合作伙伴
        </RouterLink>
        <RouterLink
          v-if="admin.can('payout.read')"
          to="/admin/payouts"
          active-class="on"
        >
          提现
        </RouterLink>
        <RouterLink
          v-if="admin.can('commission.read')"
          to="/admin/finance"
          active-class="on"
        >
          财务
        </RouterLink>
        <RouterLink
          v-if="admin.can('audit.read')"
          to="/admin/audit"
          active-class="on"
        >
          审计
        </RouterLink>
        <RouterLink
          v-if="admin.can('admin.user.manage')"
          to="/admin/administrators"
          active-class="on"
        >
          管理员
        </RouterLink>
      </nav>
      <div class="operator">
        <span>{{ admin.email || auth.displayName }}</span>
        <el-button
          size="small"
          plain
          @click="logout"
        >
          退出
        </el-button>
      </div>
    </header>

    <main class="admin-main">
      <div
        v-if="admin.status === 'loading' || auth.loading"
        v-loading="true"
        class="state-card"
      />
      <el-result
        v-else-if="admin.status === 'denied'"
        icon="warning"
        title="无管理权限"
        sub-title="当前账号没有管理后台能力。请使用管理员账号登录。"
      >
        <template #extra>
          <el-button @click="logout">
            切换账号
          </el-button>
        </template>
      </el-result>
      <el-result
        v-else-if="admin.status === 'unreachable'"
        icon="error"
        title="后台不可用"
        sub-title="无法读取管理能力，请稍后重试。"
      >
        <template #extra>
          <el-button
            type="primary"
            @click="admin.loadSession(true)"
          >
            重试
          </el-button>
        </template>
      </el-result>
      <RouterView v-else />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useSalesStore } from '@/stores/sales-store'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { preloadRouteComponents, preloadWhenIdle } from '@/utils/routes'

const admin = useAdminStore()
const auth = useAuthStore()
const router = useRouter()

onMounted(() => {
  if (admin.status === 'unknown') void admin.loadSession(true)

  // Admin pages are intentionally split. Warm their chunks once the shell is
  // interactive so the first navigation swaps data state, not an empty layout.
  preloadWhenIdle(async () => {
    await preloadRouteComponents(
      router.getRoutes().filter(route => String(route.name ?? '').startsWith('admin-')),
    )
  })
})

function clearWorkspace() {
  useEntitlementsStore().reset()
  useChatStore().reset()
  useBirthProfilesStore().reset()
  useSalesStore().reset()
  admin.reset()
}

async function logout() {
  await auth.logout()
  clearWorkspace()
  await router.push({ name: 'login' })
}
</script>

<style scoped>
.admin-shell { min-height: 100vh; min-height: 100dvh; background: var(--liki-paper); color: var(--liki-ink); }
.admin-topbar {
  min-height: 58px; display: flex; align-items: center; gap: 20px; padding: 10px 18px;
  background: var(--liki-ink); color: #fff; position: sticky; top: 0; z-index: 50;
}
.brand { font-weight: 700; font-size: 18px; letter-spacing: 1px; }
.brand span { color: var(--liki-primary); }
.admin-nav { display: flex; align-items: center; gap: 4px; flex: 1; overflow-x: auto; }
.admin-nav a {
  color: rgba(255,255,255,.76); text-decoration: none; padding: 8px 12px;
  border-radius: 8px; white-space: nowrap; font-size: 14px;
}
.admin-nav a:hover, .admin-nav a.on { background: rgba(109,90,207,.32); color: #fff; }
.operator { display: flex; align-items: center; gap: 10px; font-size: 13px; }
.admin-main { padding: 22px 18px 48px; max-width: 1360px; margin: 0 auto; }
.state-card { min-height: 240px; }
.admin-nav::-webkit-scrollbar { height: 4px; }
.admin-nav::-webkit-scrollbar-thumb { background: rgba(255,255,255,.22); border-radius: 2px; }
@media (max-width: 720px) {
  .admin-topbar { flex-wrap: wrap; gap: 10px; }
  .admin-nav { order: 3; width: 100%; }
  .admin-main { padding: 16px 12px 48px; }
}
</style>
