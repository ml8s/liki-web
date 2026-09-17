<template>
  <el-card class="skill-card" shadow="hover">
    <template #header>
      <div class="header">
        <strong>{{ label }}</strong>
        <el-tag :type="enabled ? 'success' : 'info'" size="small">
          {{ enabled ? '可用' : '待开通' }}
        </el-tag>
      </div>
    </template>

    <p>{{ description }}</p>
    <el-button
      type="primary"
      :disabled="!enabled || starting"
      :loading="starting"
      @click="$emit('start')"
    >
      开始解读
    </el-button>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  description: string
  enabled: boolean
  starting?: boolean
}>()

defineEmits<{ start: [] }>()
</script>

<style scoped>
.header { display: flex; align-items: center; justify-content: space-between; }
p { min-height: 44px; color: #555; }
</style>
