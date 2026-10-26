<template>
  <div
    class="consultation-spine"
    :data-state="state"
  >
    <section
      class="spine-context"
      :aria-labelledby="headingId"
    >
      <h1
        :id="headingId"
        class="spine-title"
      >
        {{ personLabel || domainLabel }}
      </h1>
      <p class="spine-domain">
        {{ domainLabel }}
      </p>
    </section>

    <div
      class="spine-state"
      :role="error ? 'alert' : 'status'"
      aria-live="polite"
      :aria-busy="loading"
    >
      <template v-if="loading">
        <p class="spine-visually-hidden">
          {{ $t('common.loading') }}
        </p>
        <el-skeleton
          :rows="4"
          animated
          aria-busy="true"
        />
      </template>

      <div
        v-else-if="error"
        class="spine-error"
      >
        <p>{{ error }}</p>
        <button
          type="button"
          class="spine-action"
          @click="emit('retry-profiles')"
        >
          {{ $t('common.retry') }}
        </button>
      </div>

      <AppEmptyState
        v-else-if="!hasProfile"
        :description="$t('flow.noProfiles')"
      >
        <button
          type="button"
          class="spine-action"
          @click="emit('create-profile')"
        >
          {{ $t('flow.createProfile') }}
        </button>
      </AppEmptyState>
    </div>

    <div
      v-if="state === 'ready'"
      class="spine-content"
    >
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import AppEmptyState from '@/components/app/AppEmptyState.vue'

const props = defineProps<{
  personLabel?: string
  domainLabel: string
  loading?: boolean
  error?: string
  hasProfile?: boolean
}>()

const emit = defineEmits<{
  'create-profile': []
  'retry-profiles': []
}>()

const headingId = useId()

const state = computed(() => {
  if (props.loading) return 'loading'
  if (props.error) return 'error'
  return props.hasProfile ? 'ready' : 'empty'
})
</script>

<style scoped>
.consultation-spine {
  min-width: 0;
  display: grid;
  gap: var(--space-4);
  align-content: start;
}
.spine-context {
  min-width: 0;
}
.spine-title {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--liki-ink);
  font-size: var(--font-size-title-md);
  font-weight: 650;
  line-height: 1.25;
  letter-spacing: -0.03em;
}
.spine-domain {
  margin: var(--space-1) 0 0;
  overflow-wrap: anywhere;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}
.spine-state {
  min-width: 0;
}
.spine-error p {
  margin: 0 0 var(--space-3);
  overflow-wrap: anywhere;
  color: var(--color-error);
}
.spine-action {
  width: 100%;
  min-height: var(--control-height-touch);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px var(--space-4);
  border: 0;
  border-radius: var(--radius-full);
  background: var(--liki-primary);
  color: var(--liki-on-primary);
  font: inherit;
  font-size: var(--font-size-body);
  font-weight: 600;
  cursor: pointer;
  transition: background-color var(--liki-dur) var(--liki-ease);
}
.spine-action:hover {
  background: var(--liki-primary-hover);
}
.spine-action:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.spine-content {
  min-width: 0;
}
.spine-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}
</style>
