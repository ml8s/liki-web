<template>
  <AppLiveRegion
    v-if="phase"
    class="phase"
  >
    <span
      class="dot"
      aria-hidden="true"
    />
    {{ phase }}
  </AppLiveRegion>
</template>

<script setup lang="ts">
import AppLiveRegion from '@/components/app/AppLiveRegion.vue'

defineProps<{ phase: string }>()
</script>

<style scoped>
.phase {
  max-width: 820px;
  margin: 0 auto 10px;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
.dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-inline-end: 6px;
  border-radius: 50%;
  background: var(--liki-primary);
  animation: pulse 1.2s var(--liki-ease) infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.45; transform: scale(0.8); }
}
</style>
