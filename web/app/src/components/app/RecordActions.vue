<template>
  <div class="record-actions">
    <template
      v-for="action in actions"
      :key="action.key"
    >
      <router-link
        v-if="action.to"
        :to="action.to"
        class="record-action"
        :class="{ danger: action.danger }"
        @click.stop
      >
        {{ action.label }}
      </router-link>
      <button
        v-else
        type="button"
        class="record-action"
        :class="{ danger: action.danger }"
        @click.prevent.stop="$emit('action', action.key)"
      >
        {{ action.label }}
      </button>
    </template>
  </div>
</template>

<script setup lang="ts">
/** 记录行操作的标准条目：key 用于 @action 路由，to 表示纯导航（渲染为链接）。 */
export interface RecordAction {
  key: string
  label: string
  to?: string
  danger?: boolean
}

defineProps<{ actions: RecordAction[] }>()
defineEmits<{ action: [key: string] }>()
</script>

<style scoped>
.record-actions { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: 0 2px; }
.record-action {
  display: inline-flex; align-items: center;
  min-height: 32px;
  font: inherit; font-size: var(--font-size-caption); line-height: 1;
  color: var(--liki-primary); background: none; border: 0;
  padding: 6px 8px; cursor: pointer; text-decoration: none; white-space: nowrap;
  border-radius: var(--radius-small);
  transition: opacity var(--liki-dur) var(--liki-ease);
}
.record-action:hover { opacity: 0.75; }
.record-action.danger {
  margin-left: var(--space-2);
  padding-left: var(--space-3);
  border-left: 1px solid var(--liki-line);
  border-radius: 0 var(--radius-small) var(--radius-small) 0;
  color: var(--color-error);
}
@media (max-width: 640px) {
  .record-action { min-height: 44px; padding: 10px; }
}
</style>
