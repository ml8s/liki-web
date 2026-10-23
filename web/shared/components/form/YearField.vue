<script setup lang="ts">
// 统一流年/年份输入：App 与营销岛共用同一 Element Plus 交互契约。
import { useI18n } from 'vue-i18n'
import NumberField from './fields/NumberField.vue'

withDefaults(defineProps<{
  modelValue: number
  min?: number
  max?: number
  name?: string
}>(), {
  min: 1900,
  max: 2100,
  name: 'liunian-year',
})

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const { t } = useI18n()

function updateYear(value: number | null) {
  if (value !== null) emit('update:modelValue', value)
}
</script>

<template>
  <NumberField
    :model-value="modelValue"
    :name="name"
    :label="t('profiles.chart.liunianYear')"
    :min="min"
    :max="max"
    @update:model-value="updateYear"
  />
</template>
