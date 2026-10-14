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
  status: string | number
}>()

const meta = computed(() => adminStatusMeta[String(props.status)] ?? {
  label: String(props.status),
  tag: 'info' as const,
})
</script>
