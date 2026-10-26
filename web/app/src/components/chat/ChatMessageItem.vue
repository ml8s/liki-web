<template>
  <article
    v-if="message.kind === 'system_profile'"
    class="message system"
  >
    <SystemBirthProfileCard :content="message.content" />
  </article>
  <article
    v-else
    :class="['message', message.role]"
    :data-role="message.role"
  >
    <div class="role">
      {{ $t(message.role === 'user' ? 'chat.roles.user' : 'chat.roles.assistant') }}
    </div>
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
    <div
      v-if="message.role === 'assistant' && message.error"
      class="message-error"
      role="alert"
    >
      {{ message.error }}
    </div>
    <p
      v-if="message.role === 'assistant' && message.stopped"
      class="message-stopped"
      role="status"
    >
      {{ $t('chat.stopped') }}
    </p>
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
import { defineAsyncComponent, ref } from 'vue'
import type { ChatMessage } from '@/api/chat'
import { useClipboard } from '@/composables/useClipboard'
import SystemBirthProfileCard from '@/components/chat/SystemBirthProfileCard.vue'

// Markdown rendering (markdown-it + DOMPurify) is only needed once the first
// assistant message arrives; keep the renderer out of the initial route graph.
const AppMarkdown = defineAsyncComponent(() => import('@/components/common/AppMarkdown.vue'))

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
  box-sizing: border-box;
  max-width: 820px;
  margin: 0 auto 16px;
  padding: 14px 16px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-surface);
}
.message.system { background: transparent; border: none; padding: 0; }
.message.user {
  max-width: min(78%, 640px);
  margin: 0 0 16px auto;
  background: var(--liki-primary-tint);
  border-color: var(--liki-primary-border);
}
.role {
  margin-bottom: 6px;
  font-size: var(--font-size-caption);
  font-weight: 600;
  color: var(--liki-text-2);
}
.message.user .role { text-align: right; }
.message.assistant .role { font-family: var(--liki-font-serif); font-weight: 700; color: var(--liki-primary); }
.content { line-height: 1.65; color: var(--liki-ink); overflow-wrap: anywhere; }
.message.user .content { white-space: pre-wrap; }
.message-error {
  margin: 8px 0 0;
  color: var(--color-error);
  font-size: var(--font-size-label);
}
.message-stopped {
  margin: 8px 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}
.copy {
  margin: 6px 0 0 -8px;
  padding: 4px 8px;
  border: none;
  border-radius: var(--radius-small);
  background: transparent;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
  cursor: pointer;
}
.copy:hover { color: var(--liki-primary); }

@media (max-width: 768px) {
  .copy {
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    margin: 8px 0 0 -14px;
    padding: 10px 14px;
  }
}
</style>
