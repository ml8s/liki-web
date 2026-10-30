<template>
  <FlowSheet
    :model-value="modelValue"
    :title="$t('flow.contextMenu')"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <nav
      class="context-menu"
      :aria-label="$t('flow.contextMenu')"
    >
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="context-menu__item"
        :class="{ 'is-selected': item.selected }"
        :aria-current="item.selected ? 'true' : undefined"
        @click="$emit('select', item.id)"
      >
        <span>{{ item.label }}</span>
        <span
          v-if="item.selected"
          class="context-menu__cue"
          aria-hidden="true"
        >✓</span>
      </button>
    </nav>
  </FlowSheet>
</template>

<script setup lang="ts">
import FlowSheet from './FlowSheet.vue'

defineProps<{
  modelValue: boolean
  items: ReadonlyArray<{ id: string, label: string, selected?: boolean }>
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
  select: [id: string]
}>()
</script>

<style scoped>
.context-menu {
  display: grid;
  gap: var(--space-2);
}

.context-menu__item {
  min-width: 0;
  min-height: var(--control-height-touch);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 10px var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
  background: var(--liki-surface);
  color: var(--liki-ink);
  font: inherit;
  font-size: var(--font-size-body);
  text-align: start;
  cursor: pointer;
}

.context-menu__item:hover {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}

.context-menu__item:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

.context-menu__item.is-selected {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-weight: 600;
}

.context-menu__cue {
  flex: 0 0 auto;
  font-weight: 700;
}
</style>
