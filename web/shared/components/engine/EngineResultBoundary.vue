<template>
  <div
    class="engine-result-boundary"
    :data-status="status"
    :aria-busy="refreshing || status === 'loading' ? 'true' : undefined"
  >
    <div
      v-if="status === 'loading' && !hasData"
      class="engine-result-boundary__state"
      role="status"
    >
      <slot name="loading">
        <p class="engine-result-boundary__text">
          {{ $t('common.loading') }}
        </p>
      </slot>
    </div>

    <template v-else-if="hasData">
      <div
        v-if="errorMessage"
        role="alert"
        class="engine-result-boundary__error"
      >
        <slot
          name="error"
          :error="normalizedError"
        >
          <p class="engine-result-boundary__text">
            {{ errorMessage }}
          </p>
          <button
            type="button"
            class="engine-result-boundary__retry"
            @click="$emit('retry')"
          >
            {{ $t('common.retry') }}
          </button>
        </slot>
      </div>
      <slot :result="data" />
    </template>

    <div
      v-else-if="status === 'error'"
      class="engine-result-boundary__state"
      role="alert"
    >
      <slot
        name="error"
        :error="normalizedError"
      >
        <p class="engine-result-boundary__text">
          {{ errorMessage }}
        </p>
        <button
          type="button"
          class="engine-result-boundary__retry"
          @click="$emit('retry')"
        >
          {{ $t('common.retry') }}
        </button>
      </slot>
    </div>

    <div
      v-else
      class="engine-result-boundary__state"
      role="status"
    >
      <slot name="empty" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { EngineResultFailure, EngineResultStatus } from '@shared/composables/useEngineResult'

const props = defineProps<{
  status: EngineResultStatus
  data?: unknown
  error?: EngineResultFailure | string | null
  refreshing?: boolean
}>()

defineEmits<{ retry: [] }>()

const hasData = computed(() => props.data !== null && props.data !== undefined)

const normalizedError = computed<EngineResultFailure | null>(() => {
  if (!props.error) return null
  if (props.error instanceof Error) {
    return {
      name: 'EngineResultError',
      message: props.error.message,
      cause: props.error.cause,
    }
  }
  if (typeof props.error === 'string') {
    return {
      name: 'EngineResultError',
      message: props.error,
    }
  }
  return props.error
})

const errorMessage = computed(() => normalizedError.value?.message ?? '')
</script>

<style scoped>
.engine-result-boundary {
  min-width: 0;
}
.engine-result-boundary__state {
  min-width: 0;
}
.engine-result-boundary__text {
  margin: 0 0 var(--space-3);
  color: var(--liki-text-2);
  overflow-wrap: anywhere;
}
.engine-result-boundary__error {
  margin-bottom: var(--space-3);
}
.engine-result-boundary__retry {
  min-height: var(--control-height-touch);
  padding: 10px var(--space-4);
  border: 0;
  border-radius: var(--radius-full);
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.engine-result-boundary__retry:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
</style>
