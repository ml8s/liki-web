<template>
  <el-empty :description="description" />
</template>

<script setup lang="ts">
defineProps<{ description: string }>()
</script>
