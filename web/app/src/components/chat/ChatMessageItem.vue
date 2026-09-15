<template>
  <article v-if="message.kind === 'system_profile'" class="message system">
    <SystemProfileCard :content="message.content" />
  </article>
  <article v-else :class="['message', message.role]" :data-role="message.role">
    <div class="role">{{ message.role === 'user' ? '我' : '顾问' }}</div>
    <div class="content">{{ message.content }}</div>
    <button
      v-if="message.role === 'assistant' && message.content"
      class="copy"
      type="button"
      :aria-label="copied ? '已复制' : '复制回答'"
      @click="copy"
    >
      {{ copied ? '✓ 已复制' : '复制' }}
    </button>
  </article>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ChatMessage } from '@/api/chat'
import SystemProfileCard from '@/components/chat/SystemProfileCard.vue'

const props = defineProps<{ message: ChatMessage }>()
const copied = ref(false)

// 复制：优先 navigator.clipboard，失败/不可用时降级 execCommand（借命名 chat 实践）。
async function copy() {
  const text = props.message.content
  const done = () => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 1200)
  }
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(text).then(done).catch(() => fallbackCopy(text, done))
  } else {
    fallbackCopy(text, done)
  }
}

function fallbackCopy(text: string, done: () => void) {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.top = '-9999px'
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(ta)
  if (ok) done()
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
.message.user { background: var(--liki-primary-tint); border-color: var(--liki-primary-border); }
.role { margin-bottom: 6px; font-size: 12px; font-weight: 700; color: var(--liki-text-2); }
.content { line-height: 1.65; white-space: pre-wrap; color: var(--liki-ink); }
.copy {
  margin-top: 10px;
  border: none;
  background: transparent;
  color: var(--liki-text-2);
  font-size: 12px;
  padding: 0;
  cursor: pointer;
}
.copy:hover { color: var(--liki-primary); }
</style>