<template>
  <p
    v-if="message"
    class="tool-error"
    role="alert"
  >
    {{ message }}
    <button
      v-if="retryLabel"
      type="button"
      class="tool-error__retry"
      @click="$emit('retry')"
    >
      {{ retryLabel }}
    </button>
  </p>
</template>

<script setup lang="ts">
/**
 * Standard error surface for marketing tools whose failure is not a resource
 * query (validation or a mutation such as qigua). Query failures use
 * EngineResultBoundary so they share the App's cache/retry lifecycle.
 */
defineProps<{
  message?: string
  retryLabel?: string
}>()

defineEmits<{ retry: [] }>()
</script>

<style scoped>
.tool-error__retry {
  margin-left: var(--space-2);
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}
</style>
