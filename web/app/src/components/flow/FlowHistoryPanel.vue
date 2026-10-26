<template>
  <section
    class="flow-history"
    :aria-labelledby="headingId"
  >
    <header class="flow-history__header">
      <h2 :id="headingId">
        {{ $t('flow.historyTitle') }}
      </h2>
      <AppLiveRegion class="flow-history__count">
        {{ $t('flow.historyCount', { count: filteredSessions.length }) }}
      </AppLiveRegion>
    </header>

    <TextField
      v-model="query"
      class="flow-history__search"
      name="flow-history-search"
      type="search"
      :label="$t('flow.historySearchLabel')"
      :placeholder="$t('flow.historySearchPlaceholder')"
      autocomplete="off"
    />

    <p
      v-if="error"
      class="flow-history__error"
      role="alert"
    >
      {{ error }}
    </p>

    <div
      v-if="loading && !sessions.length"
      class="flow-history__loading"
      aria-busy="true"
    >
      <el-skeleton
        :rows="3"
        animated
      />
    </div>

    <p
      v-else-if="!filteredSessions.length"
      class="flow-history__empty"
      role="status"
    >
      {{ sessions.length ? $t('flow.historyEmpty') : $t('chat.empty') }}
    </p>

    <ol
      v-else
      class="flow-history__list"
    >
      <li
        v-for="session in filteredSessions"
        :key="session.id"
        class="flow-history__item"
        :class="{ 'is-active': session.id === activeSessionId }"
      >
        <div class="flow-history__main">
          <button
            type="button"
            class="flow-history__open"
            @click="$emit('open', session)"
          >
            <span class="flow-history__title">{{ session.title }}</span>
            <span class="flow-history__meta">
              {{ formatDateTime(session.updated_at) }}
              <span
                v-if="session.id === activeSessionId"
                class="flow-history__current"
              >{{ $t('flow.historyCurrent') }}</span>
            </span>
          </button>
        </div>
        <div class="flow-history__actions">
          <button
            type="button"
            class="flow-history__action"
            @click="$emit('rename', session)"
          >
            {{ $t('chat.sessions.rename') }}
          </button>
          <button
            type="button"
            class="flow-history__action is-danger"
            @click="$emit('remove', session)"
          >
            {{ $t('chat.sessions.delete') }}
          </button>
        </div>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import type { ChatSession } from '@/api/chat'
import TextField from '@shared/components/form/fields/TextField.vue'
import AppLiveRegion from '@/components/app/AppLiveRegion.vue'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  sessions: ChatSession[]
  activeSessionId?: string | null
  loading?: boolean
  error?: string
}>()

defineEmits<{
  open: [session: ChatSession]
  rename: [session: ChatSession]
  remove: [session: ChatSession]
}>()

const headingId = useId()
const query = ref('')

const filteredSessions = computed(() => {
  const needle = query.value.trim().toLocaleLowerCase()
  if (!needle) return props.sessions
  return props.sessions.filter(session =>
    session.title.toLocaleLowerCase().includes(needle))
})
</script>

<style scoped>
.flow-history {
  min-width: 0;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
}

.flow-history__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
}

.flow-history__header h2 {
  margin: 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-xs);
  font-weight: 650;
  letter-spacing: -0.02em;
}

.flow-history__count {
  margin: 0;
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
}

.flow-history__search {
  margin-top: var(--space-3);
}

.flow-history__error,
.flow-history__empty {
  margin: var(--space-3) 0 0;
  color: var(--liki-text-2);
  overflow-wrap: anywhere;
}

.flow-history__error { color: var(--color-error); }

.flow-history__list {
  margin: var(--space-3) 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: var(--space-2);
}

.flow-history__item {
  min-width: 0;
  padding: var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-paper);
}

.flow-history__item.is-active {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}

.flow-history__main { min-width: 0; }

.flow-history__open {
  width: 100%;
  min-height: var(--control-height-touch);
  display: grid;
  gap: 2px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--liki-ink);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.flow-history__title,
.flow-history__meta {
  min-width: 0;
  overflow-wrap: anywhere;
}

.flow-history__title {
  font-weight: 600;
}

.flow-history__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}

.flow-history__current {
  color: var(--liki-primary);
  font-weight: 650;
}

.flow-history__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-1);
  margin-top: var(--space-2);
  padding-top: var(--space-2);
  border-top: 1px solid var(--liki-line);
}

.flow-history__action {
  min-height: var(--control-height-touch);
  padding: 0 var(--space-2);
  border: 0;
  background: transparent;
  color: var(--liki-primary);
  font: inherit;
  font-size: var(--font-size-caption);
  cursor: pointer;
}

.flow-history__action.is-danger { color: var(--color-error-text); }
.flow-history__action:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
  border-radius: var(--radius-small);
}

@media (min-width: 768px) {
  .flow-history__item {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--space-3);
  }

  .flow-history__actions {
    margin-top: 0;
    padding-top: 0;
    border-top: 0;
  }
}
</style>
