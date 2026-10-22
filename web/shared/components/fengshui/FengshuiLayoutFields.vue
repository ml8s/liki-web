<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElDatePicker, ElOption, ElSelect } from 'element-plus'
import { GUA_8, MOUNTAINS_24 } from '@shared/utils/xuankong'
import ElementPlusProvider from '@shared/components/form/ElementPlusProvider.vue'
import '@shared/styles/element-plus/index.css'

export interface FengshuiLayoutValue {
  zuoShan: number
  periodDate: string
  doorGua: string
  masterGua: string
  stoveGua: string
}

defineProps<{ value: FengshuiLayoutValue }>()

const emit = defineEmits<{
  'update:patch': [patch: Partial<FengshuiLayoutValue>]
  'change': []
}>()

const { t } = useI18n()

function update<K extends keyof FengshuiLayoutValue>(key: K, value: FengshuiLayoutValue[K]) {
  emit('update:patch', { [key]: value })
  emit('change')
}
</script>

<template>
  <ElementPlusProvider>
    <div class="fengshui-layout-fields">
    <el-select
      :model-value="value.zuoShan"
      popper-class="fx-popper-zuoshan"
      :aria-label="t('services.fengshui.zuoShan')"
      @update:model-value="update('zuoShan', $event as number)"
    >
      <el-option
        v-for="(name, i) in MOUNTAINS_24"
        :key="i"
        :value="i"
        :label="name"
      />
    </el-select>

    <el-date-picker
      :model-value="value.periodDate"
      type="date"
      value-format="YYYY-MM-DD"
      :aria-label="t('services.fengshui.periodDate')"
      :placeholder="t('services.fengshui.periodDate')"
      @update:model-value="update('periodDate', $event as string)"
    />

    <el-select
      :model-value="value.doorGua"
      popper-class="fx-popper-door"
      :aria-label="t('services.fengshui.door')"
      :placeholder="t('services.fengshui.selectPlaceholder')"
      @update:model-value="update('doorGua', $event as string)"
    >
      <el-option
        value=""
        disabled
        :label="t('services.fengshui.selectPlaceholder')"
      />
      <el-option
        v-for="g in GUA_8"
        :key="g"
        :value="g"
        :label="`${t('services.fengshui.door')} · ${g}`"
      />
    </el-select>

    <el-select
      :model-value="value.masterGua"
      popper-class="fx-popper-master"
      :aria-label="t('services.fengshui.master')"
      :placeholder="t('services.fengshui.selectPlaceholder')"
      @update:model-value="update('masterGua', $event as string)"
    >
      <el-option
        v-for="g in GUA_8"
        :key="g"
        :value="g"
        :label="`${t('services.fengshui.master')} · ${g}`"
      />
    </el-select>

    <el-select
      :model-value="value.stoveGua"
      popper-class="fx-popper-stove"
      :aria-label="t('services.fengshui.stove')"
      :placeholder="t('services.fengshui.selectPlaceholder')"
      @update:model-value="update('stoveGua', $event as string)"
    >
      <el-option
        v-for="g in GUA_8"
        :key="g"
        :value="g"
        :label="`${t('services.fengshui.stove')} · ${g}`"
      />
    </el-select>
    </div>
  </ElementPlusProvider>
</template>

<style scoped>
.fengshui-layout-fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
</style>