<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElDatePicker, ElFormItem, ElOption, ElSelect } from 'element-plus'
import { EVENT_KEYS } from '@shared/utils/huangli'
import ElementPlusProvider from '@shared/components/form/ElementPlusProvider.vue'
import '@shared/styles/element-plus/index.css'

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
  <ElementPlusProvider>
    <div class="huangli-range-fields">
    <el-form-item :label="t('services.huangli.event')">
      <el-select
        :model-value="value.event"
        :aria-label="t('services.huangli.event')"
        class="w-full"
        @update:model-value="update('event', $event as string)"
      >
        <el-option
          v-for="key in EVENT_KEYS"
          :key="key"
          :value="key"
          :label="t(`services.huangli.events.${key}`)"
        />
      </el-select>
    </el-form-item>

    <el-form-item
      :label="t('services.huangli.startDate')"
      prop="startDate"
    >
      <el-date-picker
        :model-value="value.startDate"
        type="date"
        value-format="YYYY-MM-DD"
        :placeholder="t('services.huangli.startDate')"
        class="w-full"
        @update:model-value="update('startDate', $event as string)"
      />
    </el-form-item>

    <el-form-item
      :label="t('services.huangli.endDate')"
      prop="endDate"
    >
      <el-date-picker
        :model-value="value.endDate"
        type="date"
        value-format="YYYY-MM-DD"
        :placeholder="t('services.huangli.endDate')"
        class="w-full"
        @update:model-value="update('endDate', $event as string)"
      />
    </el-form-item>
    </div>
  </ElementPlusProvider>
</template>

<style scoped>
.huangli-range-fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-3);
  width: 100%;
}
</style>