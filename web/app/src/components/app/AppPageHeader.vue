<template>
  <header class="app-page-header">
    <div class="copy">
      <h1>{{ title }}</h1>
      <p v-if="subtitle">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<style scoped>
.app-page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
h1 { margin: 0 0 6px; font-size: 24px; line-height: 1.25; }
p { margin: 0; color: #666; }
.actions { display: flex; align-items: center; gap: 10px; }
@media (max-width: 640px) {
  .app-page-header { align-items: flex-start; flex-direction: column; }
}
</style>
