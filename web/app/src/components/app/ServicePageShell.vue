<template>
  <section class="service-view">
    <AppPageHeader
      :title="title"
      :subtitle="subtitle"
    />

    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <slot />
  </section>
</template>

<script setup lang="ts">
import AppPageHeader from '@/components/app/AppPageHeader.vue'

defineProps<{
  title: string
  subtitle: string
  error?: string
}>()
</script>
