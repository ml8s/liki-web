<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DateField from '@shared/components/form/fields/DateField.vue'
import SelectField from '@shared/components/form/fields/SelectField.vue'
import { EVENT_KEYS } from '@shared/utils/huangli'

export interface HuangliRangeValue {
  event: string
  startDate: string
  endDate: string
}

defineProps<{ value: HuangliRangeValue }>()

const emit = defineEmits<{
  'update:patch': [patch: Partial<HuangliRangeValue>]
}>()

const { t } = useI18n()

function update<K extends keyof HuangliRangeValue>(key: K, value: HuangliRangeValue[K]) {
  emit('update:patch', { [key]: value })
}
</script>

<template>
  <div class="huangli-range-fields">
    <SelectField
      :model-value="value.event"
      name="huangli-event"
      :label="t('services.huangli.event')"
      :options="EVENT_KEYS.map(key => ({ value: key, label: t(`services.huangli.events.${key}`) }))"
      @update:model-value="update('event', $event as string)"
    />

    <DateField
      :model-value="value.startDate"
      name="huangli-start-date"
      prop="startDate"
      :label="t('services.huangli.startDate')"
      :placeholder="t('services.huangli.startDate')"
      @update:model-value="update('startDate', $event)"
    />

    <DateField
      :model-value="value.endDate"
      name="huangli-end-date"
      prop="endDate"
      :label="t('services.huangli.endDate')"
      :placeholder="t('services.huangli.endDate')"
      @update:model-value="update('endDate', $event)"
    />
  </div>
</template>

<style scoped>
.huangli-range-fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-3);
  width: 100%;
}
</style>
