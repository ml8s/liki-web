<template>
  <nav class="app-tabbar">
    <router-link to="/" exact-active-class="on">
      <span class="t-label">工作台</span>
    </router-link>
    <router-link to="/profiles" active-class="on">
      <span class="t-label">档案</span>
    </router-link>
    <router-link to="/chat" active-class="on">
      <span class="t-label">Chat</span>
    </router-link>
    <router-link to="/account" active-class="on">
      <span class="t-label">我的</span>
    </router-link>
  </nav>
</template>

<style scoped>
.app-tabbar {
  position: fixed;
  left: 0; right: 0; bottom: 0;
  display: flex;
  background: var(--liki-surface);
  border-top: 1px solid var(--liki-line);
  padding-bottom: env(safe-area-inset-bottom);
  z-index: 50;
}
.app-tabbar a {
  flex: 1;
  display: flex; align-items: center; justify-content: center;
  height: 52px;
  color: var(--liki-text-2);
  text-decoration: none;
  font-size: 14px;
  transition: color var(--liki-dur) var(--liki-ease);
}
.app-tabbar a.on { color: var(--liki-primary); font-weight: 600; }
@media (min-width: 1024px) { .app-tabbar { display: none; } }
</style>