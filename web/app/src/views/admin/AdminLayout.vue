<template>
  <div class="admin-shell">
    <a
      href="#admin-main"
      class="skip-link"
    >跳到主要内容</a>
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

    <main
      id="admin-main"
      class="admin-main"
      tabindex="-1"
    >
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
import { nextTick, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useSalesStore } from '@/stores/sales-store'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { idlePreloadAllowed, preloadRouteComponents, preloadWhenIdle } from '@/utils/routes'

const admin = useAdminStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

onMounted(() => {
  if (admin.status === 'unknown') void admin.loadSession(true)

  if (!idlePreloadAllowed()) return

  // Admin pages are intentionally split. Warm their chunks once the shell is
  // interactive so the first navigation swaps data state, not an empty layout.
  preloadWhenIdle(async () => {
    await preloadRouteComponents(
      router.getRoutes().filter(route => {
        if (!String(route.name ?? '').startsWith('admin-')) return false
        const capability = route.meta.adminCapability
        return typeof capability !== 'string' || admin.can(capability as never)
      }),
    )
  })
})

// Horizontal admin nav collapses on small operators. Keep the active item in
// view so the current route is never hidden by scroll position.
watch(() => route.fullPath, async () => {
  await nextTick()
  document.querySelector('.admin-nav a.on')?.scrollIntoView({
    inline: 'center',
    block: 'nearest',
  })
}, { immediate: true })

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
.skip-link {
  position: absolute;
  left: -9999px;
  top: 0;
  z-index: 100;
  padding: 10px 16px;
  border-radius: 0 0 8px 0;
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  text-decoration: none;
}
.skip-link:focus { left: 0; }
.admin-topbar {
  min-height: 58px; display: flex; align-items: center; gap: 20px; padding: 10px 18px;
  background: var(--liki-ink); color: var(--liki-on-ink); position: sticky; top: 0; z-index: 50;
}
.brand { font-weight: 700; font-size: 18px; letter-spacing: 1px; }
.brand span { color: var(--liki-primary); }
.admin-nav { display: flex; align-items: center; gap: 4px; flex: 1; overflow-x: auto; }
.admin-nav a {
  color: var(--liki-on-ink-subtle); text-decoration: none; padding: 8px 12px;
  border-radius: 8px; white-space: nowrap; font-size: 14px;
}
.admin-nav a:hover, .admin-nav a.on { background: var(--liki-selected-overlay); color: var(--liki-on-ink); }
.operator { display: flex; align-items: center; gap: 10px; font-size: 13px; }
.admin-main { padding: 22px 18px 48px; max-width: 1360px; margin: 0 auto; }
.state-card { min-height: 240px; }
.admin-nav::-webkit-scrollbar { height: 4px; }
.admin-nav::-webkit-scrollbar-thumb { background: var(--liki-on-ink-line); border-radius: 2px; }
@media (max-width: 720px) {
  .admin-topbar {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
  }

  .brand { order: -2; }
  .operator { order: -1; }

  .operator { justify-content: flex-end; }

  .operator span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .admin-nav {
    grid-column: 1 / -1;
    flex: 0 0 100%;
    min-width: 100%;
    scroll-snap-type: x proximity;
  }

  .admin-nav a { scroll-snap-align: start; }
  .admin-main { padding: 16px 12px 48px; }
}
</style>
