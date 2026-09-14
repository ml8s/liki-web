<template>
  <div class="app-skeleton" aria-busy="true">
    <el-skeleton :rows="rows" animated />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ rows?: number }>(), { rows: 4 })
</script>

<style scoped>
.app-skeleton { padding: 4px 0; }
</style>
