<template>
  <div v-if="phase" class="phase" role="status" aria-live="polite">
    <span class="dot" />
    {{ phase }}
  </div>
</template>

<script setup lang="ts">
defineProps<{ phase: string }>()
</script>

<style scoped>
.phase {
  max-width: 820px;
  margin: 0 auto 10px;
  color: #666;
  font-size: 12px;
}
.dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  margin-right: 6px;
  border-radius: 50%;
  background: #409eff;
}
</style>
