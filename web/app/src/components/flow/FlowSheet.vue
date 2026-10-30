<template>
  <el-drawer
    :model-value="modelValue"
    :title="title"
    class="flow-sheet"
    direction="btt"
    :size="size"
    :with-header="false"
    :append-to-body="true"
    :destroy-on-close="false"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="flow-sheet__handle">
      <span
        class="flow-sheet__grip"
        aria-hidden="true"
      />
    </div>
    <header class="flow-sheet__header">
      <h2 :id="titleId">
        {{ title }}
      </h2>
      <div
        v-if="$slots['header-actions']"
        class="flow-sheet__actions"
      >
        <slot name="header-actions" />
      </div>
      <button
        type="button"
        class="flow-sheet__close"
        :aria-label="$t('flow.close')"
        @click="$emit('update:modelValue', false)"
      >
        <span aria-hidden="true">×</span>
      </button>
    </header>
    <div
      :id="contentId"
      class="flow-sheet__body"
      role="region"
      :aria-labelledby="titleId"
    >
      <slot />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { useId } from 'vue'

defineProps<{
  modelValue: boolean
  title: string
  size?: string
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const titleId = useId()
const contentId = useId()
</script>

<style scoped>
.flow-sheet :deep(.el-drawer) {
  border-start-start-radius: var(--radius-lg);
  border-start-end-radius: var(--radius-lg);
  padding: 0;
}
.flow-sheet :deep(.el-drawer__body) {
  padding: 0 var(--space-4) calc(var(--space-4) + env(safe-area-inset-bottom));
}
.flow-sheet__handle {
  display: grid;
  place-items: center;
  min-height: var(--control-height-touch);
}
.flow-sheet__grip {
  width: 36px;
  height: 4px;
  border-radius: var(--radius-full);
  background: var(--liki-line);
}
.flow-sheet__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}
.flow-sheet__actions {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}
.flow-sheet__action {
  min-height: 32px;
  padding: 4px 10px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-text-2);
  font: inherit;
  font-size: var(--font-size-caption);
  cursor: pointer;
}
.flow-sheet__action.is-active {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
}
.flow-sheet__action:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.flow-sheet__header h2 {
  margin: 0;
  font-size: var(--font-size-title-sm);
  font-weight: 650;
  letter-spacing: -0.02em;
}
.flow-sheet__close {
  width: var(--control-height-touch);
  height: var(--control-height-touch);
  border: 0;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--liki-text-2);
  font-size: var(--font-size-title-md);
  line-height: 1;
  cursor: pointer;
}
.flow-sheet__close:hover {
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
}
.flow-sheet__close:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}
.flow-sheet__body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}
</style>

<style scoped>
@media (prefers-reduced-motion: reduce) {
  .flow-sheet :deep(.el-drawer),
  .flow-sheet :deep(.el-overlay) {
    transition: none !important;
    animation: none !important;
  }
}
</style>
