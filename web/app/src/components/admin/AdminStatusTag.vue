<template>
  <el-tag
    :type="meta.tag"
    size="small"
  >
    {{ meta.label }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { adminStatusMeta } from '@/utils/adminFormat'

const props = defineProps<{
  status: string
}>()

const meta = computed(() => adminStatusMeta[props.status] ?? {
  label: props.status,
  tag: 'info' as const,
})
</script>
