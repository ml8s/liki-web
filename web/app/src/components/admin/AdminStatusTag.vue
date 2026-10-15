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
import { adminStatusMeta, type AdminStatusDomain } from '@/utils/adminFormat'

const props = defineProps<{
  domain: AdminStatusDomain
  status: string | number
}>()

const meta = computed(() => adminStatusMeta(props.domain, String(props.status)))
</script>
