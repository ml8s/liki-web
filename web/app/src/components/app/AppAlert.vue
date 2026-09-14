<template>
  <el-alert
    class="app-alert"
    :title="title"
    :description="description"
    :type="type"
    :closable="closable"
    show-icon
    @close="$emit('close')"
  />
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  description?: string
  type?: 'success' | 'warning' | 'info' | 'error'
  closable?: boolean
}>(), {
  type: 'info',
  closable: false,
})

defineEmits<{ close: [] }>()
</script>

<style scoped>
.app-alert { margin-bottom: 16px; }
</style>
