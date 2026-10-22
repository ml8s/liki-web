<script setup lang="ts">
// 统一流年/年份输入：App 与营销岛共用同一 Element Plus 交互契约。
import { useI18n } from 'vue-i18n'
import { ElFormItem, ElInputNumber } from 'element-plus'
import ElementPlusProvider from './ElementPlusProvider.vue'
import '@shared/styles/element-plus/index.css'

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
</script>

<template>
  <ElementPlusProvider>
    <el-form-item :label="t('profiles.chart.liunianYear')">
      <el-input-number
        :model-value="modelValue"
        :min="min"
        :max="max"
        :name="name"
        controls-position="right"
        @update:model-value="emit('update:modelValue', Number($event))"
      />
    </el-form-item>
  </ElementPlusProvider>
</template>