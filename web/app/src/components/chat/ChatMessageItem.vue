<template>
  <article
    v-if="message.kind === 'system_profile'"
    class="message system"
  >
    <SystemProfileCard :content="message.content" />
  </article>
  <article
    v-else
    :class="['message', message.role]"
    :data-role="message.role"
  >
    <div class="role">
      {{ $t(message.role === 'user' ? 'chat.roles.user' : 'chat.roles.assistant') }}
    </div>
    <details
      v-if="message.role === 'assistant' && message.thinking"
      class="thinking"
      :open="!message.content"
    >
      <summary>{{ $t('chat.thinking') }}</summary>
      <div class="thinking-body">
        {{ message.thinking }}
      </div>
    </details>
    <div
      v-if="message.role === 'assistant'"
      class="content"
    >
      <AppMarkdown :source="message.content" />
    </div>
    <div
      v-else
      class="content"
    >
      {{ message.content }}
    </div>
    <button
      v-if="message.role === 'assistant' && message.content"
      class="copy"
      type="button"
      :aria-label="$t(copied ? 'chat.copied' : 'chat.copyAnswer')"
      @click="handleCopy"
    >
      {{ copied ? `✓ ${$t('chat.copied')}` : $t('common.copy') }}
    </button>
  </article>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ChatMessage } from '@/api/chat'
import { useClipboard } from '@/composables/useClipboard'
import SystemProfileCard from '@/components/chat/SystemProfileCard.vue'
import AppMarkdown from '@/components/common/AppMarkdown.vue'

const props = defineProps<{ message: ChatMessage }>()
const copied = ref(false)
const { copy } = useClipboard()

// 复制：优先 navigator.clipboard，失败/不可用时降级 execCommand
async function handleCopy() {
  await copy(props.message.content)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1200)
}
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
.message.user {
  max-width: min(78%, 640px);
  margin-left: auto;
  background: var(--liki-primary-tint);
  border-color: var(--liki-primary-border);
}
.role {
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--liki-text-2);
}
.message.user .role { text-align: right; }
.message.assistant .role { font-family: var(--liki-font-serif); font-weight: 700; color: var(--liki-primary); }
.content { line-height: 1.65; color: var(--liki-ink); }
.message.user .content { white-space: pre-wrap; }
.thinking {
  margin-bottom: 10px;
  padding: 8px 12px;
  border: 1px dashed var(--liki-line);
  border-radius: 8px;
  background: var(--liki-paper);
}
.thinking summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--liki-text-2);
  user-select: none;
}
.thinking-body {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--liki-text-2);
  white-space: pre-wrap;
}
.copy {
  margin: 6px 0 0 -8px;
  padding: 4px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--liki-text-2);
  font-size: 12px;
  cursor: pointer;
}
.copy:hover { color: var(--liki-primary); }
</style>
