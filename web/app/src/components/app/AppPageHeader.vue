<template>
  <header class="app-page-header">
    <div class="copy">
      <p
        v-if="eyebrow"
        class="eyebrow"
      >
        {{ eyebrow }}
      </p>
      <h1>{{ title }}</h1>
      <p
        v-if="subtitle"
        class="subtitle"
      >
        {{ subtitle }}
      </p>
    </div>
    <div
      v-if="$slots.actions"
      class="actions"
    >
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
  eyebrow?: string
}>()
</script>

<style scoped>
.app-page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
h1 {
  margin: 0 0 6px;
  font-size: clamp(26px, 3vw, 32px);
  font-weight: 650;
  line-height: 1.15;
  letter-spacing: -0.03em;
}
.eyebrow {
  margin: 0 0 8px;
  color: var(--liki-primary);
  font-size: 13px;
  font-weight: 600;
}
.subtitle {
  margin: 0;
  font-size: 15px;
  color: var(--liki-text-2);
  text-wrap: pretty;
}
.actions { display: flex; align-items: center; gap: 10px; }
@media (max-width: 640px) {
  .app-page-header { align-items: flex-start; flex-direction: column; }
  h1 { font-size: 27px; }
}
</style>
