<template>
  <div class="app-empty">
    <p class="app-empty__text">
      {{ description }}
    </p>
    <div
      v-if="$slots.default"
      class="app-empty__action"
    >
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
// 轻量空态（单一实现）：一句说明 + 可选主行动；不用通用插画。
defineProps<{ description: string }>()
</script>

<style scoped>
.app-empty {
  padding: var(--space-6) var(--space-4);
  text-align: center;
}
.app-empty__text {
  margin: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-body);
}
.app-empty__action {
  margin-top: var(--space-3);
}
</style>
