<template>
  <div class="thread" role="log" aria-live="polite" aria-label="消息列表">
    <AppEmpty v-if="!loading && !messages.length" description="描述你的问题，开始解读" />
    <ChatMessageItem v-for="message in messages" :key="message.id" :message="message" />
  </div>
</template>

<script setup lang="ts">
import AppEmpty from '@/components/app/AppEmpty.vue'
import ChatMessageItem from './ChatMessageItem.vue'
import type { ChatMessage } from '@/api/chat'

defineProps<{
  messages: ChatMessage[]
  loading?: boolean
}>()
</script>

<style scoped>
.thread { padding-right: 8px; }
</style>
