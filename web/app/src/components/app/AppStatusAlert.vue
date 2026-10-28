<template>
  <el-alert
    :title="title"
    :type="type"
    :closable="false"
    show-icon
    :role="type === 'error' || type === 'warning' ? 'alert' : 'status'"
    class="app-status-alert"
  >
    <div
      v-if="$slots.default"
      class="app-status-alert__content"
    >
      <slot />
    </div>
  </el-alert>
</template>

<script setup lang="ts">
import { ElAlert } from 'element-plus'

type StatusSeverity = 'info' | 'success' | 'warning' | 'error'

withDefaults(defineProps<{
  title: string
  type?: StatusSeverity
}>(), {
  type: 'info',
})

defineSlots<{
  default?: () => unknown
}>()
</script>

<style scoped>
.app-status-alert__content {
  min-width: 0;
  margin-top: var(--space-2);
}
</style>
