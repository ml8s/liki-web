<template>
  <div class="partner-card-header">
    <h2 class="title">
      <slot />
    </h2>
    <div
      v-if="$slots.actions"
      class="actions"
    >
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.partner-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.title {
  margin: 0;
  font-size: var(--font-size-body-lg);
  font-weight: 600;
  line-height: 1.35;
}

.actions {
  flex-shrink: 0;
}
</style>
