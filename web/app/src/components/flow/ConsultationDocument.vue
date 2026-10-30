<template>
  <ol
    class="consultation-document"
    :aria-label="$t('flow.documentLabel')"
    :aria-busy="loading || streaming ? 'true' : undefined"
  >
    <li
      v-if="showEmptyState"
      class="turn document-empty"
    >
      <p class="document-empty__title">
        {{ $t('flow.documentEmpty') }}
      </p>
      <p class="document-empty__hint">
        {{ $t('flow.documentEmptyHint') }}
      </p>
    </li>
    <li
      v-if="loading"
      class="turn document-loading"
      role="status"
    >
      <div class="reading">
        <el-skeleton
          :rows="4"
          animated
          aria-busy="true"
        />
      </div>
    </li>
    <li
      v-for="(message, index) in visibleMessages"
      :key="message.id"
      class="turn"
      :class="`turn--${message.role}`"
    >
      <template v-if="message.role === 'user'">
        <div class="question">
          <span class="question__label">{{ $t('flow.questionLabel') }}</span>
          <p class="question__text">
            {{ message.content }}
          </p>
        </div>
      </template>

      <template v-else>
        <article
          class="reading"
          :aria-label="$t('flow.answerLabel')"
        >
          <span class="reading__label">{{ $t('flow.answerLabel') }}</span>
          <AppMarkdown
            v-if="message.content"
            :source="message.content"
          />
          <div
            v-else-if="streaming && index === visibleMessages.length - 1"
            class="reading__loading"
            role="status"
          >
            <el-skeleton
              :rows="3"
              animated
              aria-busy="true"
            />
          </div>
          <p
            v-else-if="message.error"
            class="reading__error"
            role="alert"
          >
            {{ message.error }}
          </p>
        </article>
      </template>
    </li>
  </ol>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import type { ChatMessage } from '@/api/chat'

// Markdown rendering is only needed once the first assistant message arrives;
// keep markdown-it and its dependencies out of the initial route graph.
const AppMarkdown = defineAsyncComponent(() => import('@/components/common/AppMarkdown.vue'))

const props = defineProps<{
  messages: readonly ChatMessage[]
  streaming?: boolean
  loading?: boolean
}>()

const visibleMessages = computed(() =>
  props.messages.filter(message => message.kind === 'text'),
)

const showEmptyState = computed(() =>
  !props.loading && !props.streaming && props.messages.length === 0,
)
</script>

<style scoped>
.consultation-document {
  display: grid;
  gap: var(--space-4);
  margin: 0;
  padding: 0;
  list-style: none;
}
.document-loading { list-style: none; }
.document-loading .reading { min-height: 7rem; }
.document-empty { list-style: none; }
.document-empty__title {
  margin: 0 0 var(--space-2);
  color: var(--liki-ink);
  font-size: var(--font-size-body);
  font-weight: 600;
}
.document-empty__hint {
  margin: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-body);
}
.reading :deep(.markdown) {
  overflow-wrap: anywhere;
}
.reading :deep(img) {
  max-width: 100%;
  height: auto;
}
.reading :deep(pre) {
  min-width: 0;
}
.reading :deep(pre code) {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.reading :deep(table) {
  table-layout: fixed;
}
.reading :deep(th),
.reading :deep(td) {
  overflow-wrap: anywhere;
}
.question {
  max-width: 680px;
  margin-inline: auto;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--liki-primary-border);
  border-radius: var(--radius-lg);
  background: var(--liki-primary-tint);
}
.question__label,
.reading__label {
  display: block;
  margin-bottom: var(--space-1);
  color: var(--liki-text-3);
  font-size: var(--font-size-micro);
  font-weight: 650;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.question__text {
  margin: 0;
  color: var(--liki-ink);
  overflow-wrap: anywhere;
}
.reading {
  min-width: 0;
  max-width: min(100%, 760px);
  margin-inline: auto;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-surface);
}
.reading :deep(.markdown) {
  overflow-wrap: anywhere;
}
.reading__loading,
.reading__error {
  margin: 0;
}
.reading__error { color: var(--color-error); }
@media (min-width: 768px) {
  .turn--user { padding-inline: var(--space-6); }
}
</style>
