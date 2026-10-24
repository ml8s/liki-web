<script setup lang="ts">
// naming 页免费用神查询岛：出生信息 → bazi_chart（engine）→ bazi_yongshen（counsel）。
// 只展示用神方向（指导起名）；名字生成在 App 侧进行——公开工具层只暴露确定性裁决。
import { reactive, ref } from 'vue'
import { namingAssessmentFor, type NamingAssessment } from '@shared/domain/naming'
import { baziChartFor } from '@shared/domain/natal'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import YongShenPanel from '@shared/components/charts/YongShenPanel.vue'
import { consultationBirthFromFields, type ConsultationContext } from '@shared/domain/consultation'
import { HANDOFF_KEYS, saveConsultationHandoff, saveHandoff } from '@shared/utils/handoff'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
    computing: string
    error: string
    note: string
    cta: string
    privacy: string
    }
}>()

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const loading = ref(false)
const failed = ref(false)
const result = ref<NamingAssessment | null>(null)

async function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  loading.value = true
  failed.value = false
  result.value = null
  try {
    const chart = await baziChartFor(birthInputFromFields(birth))
    result.value = await namingAssessmentFor(chart)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function continueToApp() {
  if (!result.value) return
  saveConsultationHandoff({
    version: 1,
    scenario: 'naming',
    birth: consultationBirthFromFields(birth),
  } satisfies ConsultationContext)
  saveHandoff(HANDOFF_KEYS.birth, { ...birth })
}
</script>

<template>
  <FormShell
    form-class="tool-form"
    :model="birth"
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <BirthFields :value="birth" name-prefix="naming-birth" :labels="labels" @update:patch="Object.assign(birth, $event)" />
  </FormShell>

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="result">
    <div class="tool-pan">
      <YongShenPanel :result="result" />
    </div>
    <p class="tool-privacy">{{ labels.note }}</p>
    <a
      class="btn-purple tool-cta"
      :href="appUrl"
      @click="continueToApp"
    >{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
