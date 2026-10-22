<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElFormItem, ElInputNumber, ElOption, ElSelect } from 'element-plus'
import ElementPlusProvider from '@shared/components/form/ElementPlusProvider.vue'
import '@shared/styles/element-plus/index.css'

export interface FengshuiBirthValue {
  year: number | null
  gender: '' | 'female' | 'male'
}

const props = defineProps<{ value: FengshuiBirthValue }>()

const emit = defineEmits<{
  'update:patch': [patch: Partial<FengshuiBirthValue>]
}>()

const { t } = useI18n()

function update<K extends keyof FengshuiBirthValue>(key: K, value: FengshuiBirthValue[K]) {
  emit('update:patch', { [key]: value })
}
</script>

<template>
  <ElementPlusProvider>
    <div class="fengshui-birth-fields">
    <el-form-item :label="t('profiles.form.birthYear')">
      <el-input-number
        :model-value="value.year"
        :min="1900"
        :max="2100"
        :aria-label="t('profiles.form.birthYear')"
        @update:model-value="update('year', $event ?? null)"
      />
    </el-form-item>
    <el-form-item :label="t('profiles.form.gender')">
      <el-select
        :model-value="value.gender"
        :aria-label="t('profiles.form.gender')"
        :placeholder="t('profiles.form.gender')"
        @update:model-value="update('gender', $event as FengshuiBirthValue['gender'])"
      >
        <el-option
          value=""
          disabled
          :label="t('profiles.form.gender')"
        />
        <el-option
          value="female"
          :label="t('profiles.female')"
        />
        <el-option
          value="male"
          :label="t('profiles.male')"
        />
      </el-select>
    </el-form-item>
    </div>
  </ElementPlusProvider>
</template>

<style scoped>
.fengshui-birth-fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
</style>