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
        >
          {{ $t('nav.workspace') }}
        </router-link>
        <router-link
          to=" /birth-profiles"
          active-class="on"
        >
          {{ $t('nav.profiles') }}
        </router-link>
        <router-link
          to="/chat"
          active-class="on"
        >
          {{ $t('nav.chat') }}
        </router-link>
      </div>
      <div class="nav-group nav-group-secondary">
        <div class="nav-label">
          {{ $t('nav.account') }}
        </div>
        <router-link
          to="/orders"
          active-class="on"
        >
          {{ $t('nav.orders') }}
        </router-link>
        <router-link
          to="/account"
          active-class="on"
        >
          {{ $t('nav.account') }}
        </router-link>
      </div>
      <div class="nav-group nav-group-secondary">
        <div class="nav-label">
          {{ $t('nav.partner') }}
        </div>
        <router-link
          to="/partners"
          exact-active-class="on"
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
defineProps<{ email: string }>()
defineEmits<{ logout: [] }>()
</script>

<style scoped>
.app-sidebar {
  position: fixed;
  inset: 0 auto 0 0;
  width: 240px;
  display: flex;
  flex-direction: column;
  background: var(--liki-ink);
  color: #fff;
  padding: 24px 16px;
  z-index: 50;
}
.side-brand { font-size: 20px; font-weight: 700; letter-spacing: 2px; color: #fff; text-decoration: none; }
.side-brand em { color: var(--liki-primary); font-style: normal; }
.side-slogan { font-size: 12px; color: #c0c5cc; margin: 6px 0 24px; }
.side-nav { display: flex; flex-direction: column; gap: 20px; flex: 1; }
.nav-group { display: flex; flex-direction: column; gap: 4px; }
.nav-group-secondary { border-top: 1px solid rgba(255,255,255,.12); padding-top: 16px; }
.nav-label { font-size: 12px; color: #8b93a1; letter-spacing: 1px; padding: 0 12px 4px; }
.side-nav a {
  position: relative;
  color: #cfd3db; text-decoration: none; padding: 10px 12px;
  border-radius: 8px; font-size: 15px;
  transition: background var(--liki-dur) var(--liki-ease), color var(--liki-dur) var(--liki-ease);
}
.side-nav a:hover { background: rgba(255,255,255,.08); color: #fff; }
.side-nav a.on { background: rgba(109,90,207,.32); color: #fff; }
.side-nav a.on::before {
  content: '';
  position: absolute;
  left: 0; top: 8px; bottom: 8px;
  width: 2px;
  border-radius: 2px;
  background: var(--liki-primary);
}
.side-user { border-top: 1px solid rgba(255,255,255,.12); padding-top: 16px; display: flex; flex-direction: column; gap: 10px; }
.email { font-size: 13px; opacity: .85; word-break: break-all; }
@media (max-width: 1023px) { .app-sidebar { display: none; } }
</style>
