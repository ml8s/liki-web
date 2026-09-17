<template>
  <!-- 桌面：表格 -->
  <el-table v-if="!isMobile" :data="sessions" @row-click="$emit('open', $event)">
    <el-table-column prop="title" label="主题" min-width="180" />
    <el-table-column label="领域" width="140">
      <template #default="{ row }">{{ row.skillLabel }}</template>
    </el-table-column>
    <el-table-column label="更新时间" width="190">
      <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
    </el-table-column>
    <el-table-column width="100" align="right">
      <template #default="{ row }">
        <router-link :to="`/chat/${row.id}`" class="open-link">打开</router-link>
      </template>
    </el-table-column>
  </el-table>

  <!-- 移动端：卡片降级 -->
  <div v-else class="session-cards">
    <router-link v-for="s in sessions" :key="s.id" :to="`/chat/${s.id}`" class="session-card">
      <div class="sc-top">
        <span class="sc-title">{{ s.title }}</span>
        <el-tag size="small" effect="light">{{ s.skillLabel }}</el-tag>
      </div>
      <div class="sc-time">{{ formatDateTime(s.updated_at) }}</div>
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { ChatSession } from '@/api/chat'
import { formatDateTime } from '@/utils/format'

defineProps<{
  sessions: Array<ChatSession & { skillLabel?: string }>
}>()

defineEmits<{ open: [session: ChatSession] }>()

const isMobile = ref(false)
let mq: MediaQueryList | null = null
function onMqChange(e: MediaQueryListEvent | MediaQueryList) {
  isMobile.value = e.matches
}
onMounted(() => {
  mq = window.matchMedia('(max-width: 640px)')
  onMqChange(mq)
  mq.addEventListener('change', onMqChange)
})
onBeforeUnmount(() => mq?.removeEventListener('change', onMqChange))
</script>

<style scoped>
.open-link { display: inline-block; padding: 4px 2px; }
.session-cards { display: grid; gap: 12px; }
.session-card {
  display: block;
  background: var(--liki-surface);
  border: 1px solid var(--liki-line);
  border-radius: 12px;
  padding: 14px 16px;
  text-decoration: none;
  color: inherit;
  transition: border-color var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease), box-shadow var(--liki-dur) var(--liki-ease);
}
.session-card:hover {
  border-color: var(--liki-primary-border);
  transform: translateY(-1px);
  box-shadow: 0 6px 18px rgba(30, 34, 48, 0.08);
}
.sc-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.sc-title { font-weight: 600; }
.sc-time { color: var(--liki-text-2); font-size: 13px; margin-top: 6px; }
</style>