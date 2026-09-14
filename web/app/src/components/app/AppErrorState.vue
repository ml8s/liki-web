<template>
  <el-result icon="warning" :title="title" :sub-title="description">
    <template #extra>
      <el-button v-if="retryable" type="primary" @click="$emit('retry')">重试</el-button>
    </template>
  </el-result>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
  retryable?: boolean
}>(), {
  title: '加载失败',
  description: '请稍后重试',
  retryable: true,
})

defineEmits<{ retry: [] }>()
</script>
