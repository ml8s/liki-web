<template>
  <div
    class="thread"
    role="log"
    aria-live="polite"
    :aria-label="$t('chat.threadLabel')"
  >
    <div
      v-if="!loading && !messages.length"
      class="empty-with-suggestions"
    >
      <el-empty
        :description="$t('chat.emptyThread')"
      />
      <div
        class="suggestions"
        role="group"
        :aria-label="$t('chat.suggestionsLabel')"
      >
        <button
          v-for="suggestion in suggestions"
          :key="suggestion"
          type="button"
          class="suggestion"
          @click="$emit('select', suggestion)"
        >
          {{ suggestion }}
        </button>
      </div>
    </div>
    <ChatMessageItem
      v-for="message in messages"
      :key="message.id"
      :message="message"
    />
  </div>
</template>

<script setup lang="ts">
import ChatMessageItem from './ChatMessageItem.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatMessage } from '@/api/chat'

defineProps<{
  messages: ChatMessage[]
  loading?: boolean
}>()

defineEmits<{ select: [prompt: string] }>()
const { t } = useI18n()

const suggestions = computed(() => [
  t('chat.suggestions.career'),
  t('chat.suggestions.relationship'),
  t('chat.suggestions.yearlyChanges'),
])
</script>

<style scoped>
.empty-with-suggestions {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.suggestions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  max-width: 640px;
  margin-top: -18px;
}
.suggestion {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 10px 16px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
  cursor: pointer;
  transition: border-color var(--liki-dur) var(--liki-ease), color var(--liki-dur) var(--liki-ease);
}
.suggestion:hover {
  border-color: var(--liki-primary-border);
  color: var(--liki-primary);
}
</style>
