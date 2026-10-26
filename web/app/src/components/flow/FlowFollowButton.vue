<template>
  <button
    type="button"
    class="flow-follow"
    @click="$emit('follow')"
  >
    {{ label }}
  </button>
</template>

<script setup lang="ts">
defineProps<{
  label: string
}>()

defineEmits<{
  follow: []
}>()
</script>

<style scoped>
.flow-follow {
  min-width: 0;
  max-width: 100%;
  min-height: var(--control-height-touch);
  padding-inline: var(--space-3);
  border: 1px solid var(--liki-primary-border);
  border-radius: var(--radius-full);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font: inherit;
  font-size: var(--font-size-label);
  font-weight: 600;
  overflow-wrap: anywhere;
  text-wrap: balance;
  cursor: pointer;
}

.flow-follow:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
</style>
