<script setup lang="ts">
// naming 页免费用神查询岛：出生信息 → bazi_chart（engine）→ bazi_yongshen（counsel）。
// 只展示用神方向（指导起名）；名字生成在 App 侧进行——公开工具层只暴露确定性裁决。
import { computed, reactive, ref } from 'vue'
import { namingAssessmentFor, type NamingAssessment } from '@shared/domain/naming'
import { baziChartFor } from '@shared/domain/natal'
import type { BirthInput } from '@shared/domain/birth'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import BaziYongShenPanel from '@shared/components/engine/bazi/BaziYongShenPanel.vue'
import { birthFormRules } from '@shared/components/form/birth-rules'
import { consultationBirthFromFields, type ConsultationContext } from '@shared/domain/consultation'
import { HANDOFF_KEYS, saveConsultationHandoff, saveHandoff } from '@shared/utils/handoff'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
    computing: string
    error: string
    resultIntro: string
    note: string
    cta: string
    privacy: string
    }
}>()

const rules = birthFormRules(props.labels)
const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const birthInput = ref<BirthInput | null>(null)
const status = ref<EngineResultStatus>('idle')
const result = ref<NamingAssessment | null>(null)

const loading = computed(() =>
  !!birthInput.value && (status.value === 'idle' || status.value === 'loading'),
)

function onData(value: unknown) {
  result.value = (value as NamingAssessment | null) ?? null
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function loadNaming(input: BirthInput) {
  return baziChartFor(input).then(chart => namingAssessmentFor(chart))
}

function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  result.value = null
  birthInput.value = birthInputFromFields(birth)
}

function continueToApp() {
  if (!birthInput.value) return
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
    :rules="rules"
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <BirthFields :value="birth" name-prefix="naming-birth" :labels="labels" @update:patch="Object.assign(birth, $event)" />
  </FormShell>

  <EngineQueryPanel
    v-if="birthInput"
    resource="naming.assessment"
    :input="birthInput"
    :load="loadNaming"
    :enabled="!!birthInput"
    :fallback="labels.error"
    @update:data="onData"
    @update:status="onStatus"
  >
    <template #default="{ result: assessment }">
      <div class="tool-pan">
        <BaziYongShenPanel :result="assessment" />
      </div>
      <p class="tool-pan-intro">
        {{ labels.resultIntro }}
      </p>
      <p class="tool-privacy">{{ labels.note }}</p>
      <ToolCta
        :href="appUrl"
        :privacy="labels.privacy"
        @click="continueToApp"
      >
        {{ labels.cta }}
      </ToolCta>
    </template>
  </EngineQueryPanel>
</template>
