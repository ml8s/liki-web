<script setup lang="ts">
// 统一出生信息字段：完整信息才可排盘；无 approximate / unknown 分支。
import {
  ElDatePicker, ElFormItem, ElRadioButton, ElRadioGroup, ElTimePicker,
} from 'element-plus'
import RegionField from './RegionField.vue'
import ElementPlusProvider from './ElementPlusProvider.vue'
import type {
  BirthFieldLabels, BirthFieldsValue,
} from './birth-fields'
import type { BirthGender } from '@shared/domain/birth'
import '@shared/styles/element-plus/index.css'

const props = withDefaults(defineProps<{
  value: BirthFieldsValue
  labels: BirthFieldLabels
  namePrefix?: string
  showRegion?: boolean
  initialPlace?: string
}>(), {
  namePrefix: 'birth',
  showRegion: true,
  initialPlace: '',
})

const emit = defineEmits<{
  'update:patch': [patch: Partial<BirthFieldsValue>]
}>()

function update<K extends keyof BirthFieldsValue>(key: K, value: BirthFieldsValue[K]) {
  emit('update:patch', { [key]: value })
}

function disabledDate(date: Date) {
  return date.getTime() > Date.now() || date.getFullYear() < 1900
}
</script>

<template>
  <ElementPlusProvider>
    <el-form-item
      :label="props.labels.gender"
      prop="gender"
    >
      <el-radio-group
        :model-value="value.gender"
        :name="`${namePrefix}-gender`"
        @update:model-value="update('gender', $event as BirthGender)"
      >
        <el-radio-button value="female">
          {{ props.labels.female }}
        </el-radio-button>
        <el-radio-button value="male">
          {{ props.labels.male }}
        </el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item
      :label="props.labels.birthDate"
      prop="date"
    >
      <el-date-picker
        :model-value="value.date"
        type="date"
        value-format="YYYY-MM-DD"
        :aria-label="props.labels.birthDate"
        :placeholder="props.labels.selectDate"
        :disabled-date="disabledDate"
        class="w-full"
        @update:model-value="update('date', $event)"
      />
    </el-form-item>

    <el-form-item
      :label="props.labels.birthTime"
      prop="time"
    >
      <el-time-picker
        :model-value="value.time"
        :aria-label="props.labels.birthTime"
        format="HH:mm"
        value-format="HH:mm"
        :placeholder="props.labels.timePlaceholder"
        class="w-full"
        @update:model-value="update('time', $event)"
      />
    </el-form-item>

    <slot />

    <RegionField
      v-if="showRegion"
      :location="value.location"
      :labels="props.labels.region"
      :name-prefix="namePrefix"
      :initial-place="initialPlace"
      @update:location="update('location', $event)"
    />
  </ElementPlusProvider>
</template>
