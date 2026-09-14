<template>
  <article v-if="message.kind === 'system_profile'" class="message system">
    <SystemProfileCard :content="message.content" />
  </article>
  <article v-else :class="['message', message.role]" :data-role="message.role">
    <div class="role">{{ message.role === 'user' ? '我' : '顾问' }}</div>
    <div class="content">{{ message.content }}</div>
  </article>
</template>

<script setup lang="ts">
import type { ChatMessage } from '@/api/chat'
import SystemProfileCard from '@/components/chat/SystemProfileCard.vue'

defineProps<{ message: ChatMessage }>()
</script>

<style scoped>
.message {
  max-width: 820px;
  margin: 0 auto 16px;
  padding: 14px 16px;
  border: 1px solid var(--liki-line);
  border-radius: 12px;
  background: var(--liki-surface);
}
.message.system { background: transparent; border: none; padding: 0; }
.message.user { background: var(--liki-primary-tint); border-color: var(--liki-primary-border); }
.role { margin-bottom: 6px; font-size: 12px; font-weight: 700; color: var(--liki-text-2); }
.content { line-height: 1.65; white-space: pre-wrap; color: var(--liki-ink); }
</style>