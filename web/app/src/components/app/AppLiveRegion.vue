<template>
  <div
    :role="role"
    :aria-live="level"
    aria-atomic="true"
    :class="{ 'app-live-region--screen-only': screenOnly }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  level?: 'polite' | 'assertive'
  role?: 'status' | 'alert'
  screenOnly?: boolean
}>(), {
  level: 'polite',
  role: 'status',
  screenOnly: false,
})

defineSlots<{ default?: () => unknown }>()
</script>

<style scoped>
.app-live-region--screen-only {
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
