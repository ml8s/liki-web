<template>
  <div class="app-empty">
    <el-empty :description="description" />
    <p class="app-empty-slogan font-serif">懂命理，用靈機</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{ description: string }>()
</script>

<style scoped>
.app-empty-slogan { color: var(--liki-text-3); font-size: 13px; margin: -4px 0 8px; }
</style>