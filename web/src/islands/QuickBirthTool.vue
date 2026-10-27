<script setup lang="ts">
// 首页免费排盘：与 natal 页共用同一个排盘岛，避免首页只做交接。
// 免费盘在营销站完成；只有点击“解读”才把手交接给 App。
import NatalTool from './NatalTool.vue'
import type { BirthFieldLabels } from '@shared/components/form/birth-fields'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
    computing: string
    error: string
    bazi: string
    ziwei: string
    lucky: string
    cta: string
    privacy: string
  }
}>()
</script>

<template>
  <NatalTool v-bind="props" />
</template>
