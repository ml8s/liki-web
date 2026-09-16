<template>
  <aside class="app-sidebar">
    <router-link to="/" class="side-brand font-serif">Liki<em>·</em>命理</router-link>
    <div class="side-slogan font-serif">懂命理，用 Liki</div>

    <nav class="side-nav">
      <router-link to="/" exact-active-class="on">工作台</router-link>
      <router-link to="/profiles" active-class="on">档案</router-link>
      <router-link to="/chat" active-class="on">Chat</router-link>
      <router-link to="/orders" active-class="on">订单</router-link>
      <router-link to="/account" active-class="on">账户</router-link>
    </nav>

    <div class="side-user">
      <span class="email">{{ email }}</span>
      <el-button size="small" plain @click="$emit('logout')">退出</el-button>
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
.side-brand em { color: var(--liki-gold); font-style: normal; }
.side-slogan { font-size: 12px; color: #c0c5cc; margin: 6px 0 24px; }
.side-nav { display: flex; flex-direction: column; gap: 4px; flex: 1; }
.side-nav a {
  color: #cfd3db; text-decoration: none; padding: 10px 12px;
  border-radius: 6px; font-size: 15px;
  transition: background var(--liki-dur) var(--liki-ease), color var(--liki-dur) var(--liki-ease);
}
.side-nav a:hover { background: rgba(255,255,255,.08); }
.side-nav a.on { background: var(--liki-primary); color: #fff; }
.side-user { border-top: 1px solid rgba(255,255,255,.12); padding-top: 16px; display: flex; flex-direction: column; gap: 10px; }
.email { font-size: 13px; opacity: .85; word-break: break-all; }
@media (max-width: 1023px) { .app-sidebar { display: none; } }
</style>