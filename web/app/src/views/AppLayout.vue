<template>
  <div
    class="app-shell"
    :class="{ 'app-shell-bounded': viewportBound }"
    :style="shellHeightStyle"
  >
    <a
      href="#app-main"
      class="skip-link"
    >{{ $t('a11y.skipToContent') }}</a>
    <AppSidebar
      :email="auth.user?.email ?? ''"
      @logout="logout"
    />

    <header class="app-topbar">
      <div class="brand font-serif">
        Liki<em>·</em>{{ $t('auth.brandTail') }}
      </div>
      <div class="topbar-meta">
        <span class="page-title">{{ pageTitle }}</span>
        <el-button
          size="small"
          link
          @click="logout"
        >
          {{ $t('nav.logout') }}
        </el-button>
      </div>
    </header>

    <main
      id="app-main"
      ref="mainEl"
      class="app-main"
      tabindex="-1"
    >
      <router-view />
    </main>

    <AppTabbar />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import AppSidebar from '@/components/app/AppSidebar.vue'
import AppTabbar from '@/components/app/AppTabbar.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const mainEl = ref<HTMLElement | null>(null)
const pageTitle = computed(() => route.meta.titleKey ? t(String(route.meta.titleKey)) : t('router.dashboard'))
// 内容页保持文档滚动（URL 栏收起、页内查找、下拉刷新都是平台行为）；
// 只有自称独占视口的视图才把滚动收进页面内部（如会话页的消息区）。
const viewportBound = computed(() => route.meta.viewportBound === true)

// 移动端键盘开合时 visualViewport 缩小而 dvh 不变：独占视口的页面跟随可视高度，
// 保证 composer 不被键盘遮挡（内容页依赖平台自身行为，不介入）。
const vvHeight = ref<number | null>(null)
const shellHeightStyle = computed(() =>
  viewportBound.value && vvHeight.value ? { height: `${vvHeight.value}px` } : undefined,
)

function syncViewportHeight() {
  const vv = window.visualViewport
  vvHeight.value = vv ? Math.round(vv.height) : null
}

onMounted(async () => {
  try {
    await auth.loadMe()
  } catch {
    // 路由守卫已处理 401 重定向；网络异常不崩溃整页。
  }
  if (window.visualViewport) {
    window.visualViewport.addEventListener('resize', syncViewportHeight)
    syncViewportHeight()
  }
})

onBeforeUnmount(() => {
  window.visualViewport?.removeEventListener('resize', syncViewportHeight)
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
.app-shell {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--liki-paper);
}
.app-shell-bounded {
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
}

.skip-link {
  position: absolute; left: -9999px; top: 0; z-index: var(--z-skip-link);
  background: var(--liki-primary); color: var(--liki-on-primary);
  padding: 10px 16px; border-radius: 0 0 var(--radius-md) 0; text-decoration: none;
}
.skip-link:focus { left: 0; }

.app-topbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; background: var(--liki-ink); color: var(--liki-on-ink);
  position: sticky; top: 0; z-index: var(--z-header);
}
.app-topbar .brand { font-size: var(--font-size-title-sm); font-weight: 700; letter-spacing: 2px; }
.app-topbar .brand em { color: var(--liki-primary); font-style: normal; }
.topbar-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.page-title {
  overflow: hidden;
  color: var(--liki-on-ink-subtle);
  font-size: var(--font-size-body);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.topbar-meta .el-button {
  color: var(--liki-on-ink);
  --el-button-text-color: var(--liki-on-ink);
  --el-button-hover-text-color: var(--liki-on-ink);
}

.app-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 24px 16px 88px;
}
.app-main:focus-visible { outline: 2px solid var(--liki-primary); outline-offset: 2px; }

@media (min-width: 1024px) {
  .app-topbar { display: none; }
  /* 240px 侧栏之后，剩余空间里居中内容列，避免超宽屏贴左。 */
  .app-main { margin-left: max(240px, calc(50vw - 520px)); padding: 32px 40px 48px; max-width: 1200px; }
}

@media (max-width: 640px) {
  .app-main { padding: 18px 14px 88px; }
  .page-title { display: none; }
}
</style>
