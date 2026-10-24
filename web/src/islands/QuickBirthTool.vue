<script setup lang="ts">
// 首页免费排盘只是交接宿主：字段展示与 App / 营销岛共用同一个 BirthFields。
import { reactive } from 'vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import {
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import { birthFormRules } from '@shared/components/form/birth-rules'
import { saveHandoff, HANDOFF_KEYS } from '@shared/utils/handoff'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
  }
}>()

const rules = birthFormRules(props.labels)
const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())

function submit() {
  if (!isBirthInputReady(birth)) return
  saveHandoff(HANDOFF_KEYS.birth, { ...birth })
  window.location.href = props.appUrl
}
</script>

<template>
  <FormShell
    form-class="tool-form"
    :model="birth"
    :rules="rules"
    :submit-label="labels.submit"
    @submit="submit"
  >
    <BirthFields
      :value="birth"
      :labels="labels"
      @update:patch="Object.assign(birth, $event)"
    />
  </FormShell>
</template>
