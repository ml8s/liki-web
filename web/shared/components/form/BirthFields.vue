<script setup lang="ts">
// 统一出生信息字段：完整信息才可排盘；无 approximate / unknown 分支。
import { computed } from 'vue'
import DateField from './fields/DateField.vue'
import TimeField from './fields/TimeField.vue'
import ChoiceCardGroup from './fields/ChoiceCardGroup.vue'
import RegionAutocompleteField from './fields/RegionAutocompleteField.vue'
import './styles/field.css'
import type {
  BirthFieldLabels, BirthFieldsValue,
} from './birth-fields'
import type { BirthGender } from '@shared/domain/birth'

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

const genderOptions = computed(() => [
  { value: 'female' as const, label: props.labels.female },
  { value: 'male' as const, label: props.labels.male },
])

function update<K extends keyof BirthFieldsValue>(key: K, value: BirthFieldsValue[K]) {
  emit('update:patch', { [key]: value })
}

function disabledDate(date: Date) {
  return date.getTime() > Date.now() || date.getFullYear() < 1900
}
</script>

<template>
  <ChoiceCardGroup
    :model-value="value.gender"
    :name="`${namePrefix}-gender`"
    :label="props.labels.gender"
    prop="gender"
    size="lg"
    :options="genderOptions"
    @update:model-value="update('gender', $event as BirthGender)"
  />

  <DateField
    :model-value="value.date"
    :name="`${namePrefix}-date`"
    prop="date"
    :label="props.labels.birthDate"
    :placeholder="props.labels.selectDate"
    :disabled-date="disabledDate"
    @update:model-value="update('date', $event)"
  />

  <TimeField
    :model-value="value.time"
    :name="`${namePrefix}-time`"
    prop="time"
    :label="props.labels.birthTime"
    :placeholder="props.labels.timePlaceholder"
    @update:model-value="update('time', $event)"
  />

  <slot />

  <div
    v-if="showRegion"
    class="lf-field"
  >
    <RegionAutocompleteField
      :location="value.location"
      :labels="props.labels.region"
      :name-prefix="namePrefix"
      :initial-place="initialPlace"
      @update:location="update('location', $event)"
    />
  </div>
</template>
