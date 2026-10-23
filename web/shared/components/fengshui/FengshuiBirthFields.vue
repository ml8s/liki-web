<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ChoiceCardGroup from '@shared/components/form/fields/ChoiceCardGroup.vue'
import NumberField from '@shared/components/form/fields/NumberField.vue'

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
  <div class="fengshui-birth-fields">
    <NumberField
      :model-value="value.year"
      name="fengshui-birth-year"
      :label="t('profiles.form.birthYear')"
      :min="1900"
      :max="2100"
      @update:model-value="update('year', $event)"
    />
    <ChoiceCardGroup
      :model-value="value.gender"
      name="fengshui-gender"
      :label="t('profiles.form.gender')"
      :options="[
        { value: 'female', label: t('profiles.female') },
        { value: 'male', label: t('profiles.male') },
      ]"
      @update:model-value="update('gender', $event as FengshuiBirthValue['gender'])"
    />
  </div>
</template>

<style scoped>
.fengshui-birth-fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
</style>
