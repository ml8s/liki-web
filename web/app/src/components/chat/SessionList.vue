<template>
  <el-table :data="sessions" @row-click="$emit('open', $event)">
    <el-table-column prop="title" label="主题" min-width="180" />
    <el-table-column label="领域" width="140">
      <template #default="{ row }">{{ row.skillLabel }}</template>
    </el-table-column>
    <el-table-column prop="updated_at" label="更新时间" width="190" />
    <el-table-column width="100" align="right">
      <template #default="{ row }">
        <router-link :to="`/chat/${row.id}`" class="open-link">打开</router-link>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { ChatSession } from '@/api/chat'

defineProps<{
  sessions: Array<ChatSession & { skillLabel?: string }>
}>()

defineEmits<{ open: [session: ChatSession] }>()
</script>

<style scoped>
.open-link { display: inline-block; padding: 4px 2px; }
</style>
