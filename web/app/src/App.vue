<template>
  <div v-if="fatal.crashed" class="fatal" role="alert">
    <h1>页面出了点问题</h1>
    <p class="msg">{{ fatal.message || '发生了意外错误。' }}</p>
    <div class="actions">
      <el-button @click="retry">返回首页</el-button>
      <el-button type="primary" @click="reload">重新加载</el-button>
    </div>
  </div>
  <router-view v-else v-slot="{ Component }">
    <transition name="page-fade" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { fatal } from '@/errors'

const router = useRouter()

function reload() {
  window.location.reload()
}

function retry() {
  fatal.crashed = false
  fatal.message = ''
  void router.push('/')
}
</script>

<style scoped>
.fatal {
  min-height: 100vh;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  background: var(--liki-paper);
  text-align: center;
  padding: 24px;
}
.fatal h1 { font-size: 22px; margin: 0; }
.fatal .msg { color: var(--liki-text-2); max-width: 440px; margin: 0; }
.actions { display: flex; gap: 12px; margin-top: 8px; }
</style>