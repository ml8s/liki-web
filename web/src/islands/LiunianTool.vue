<script setup lang="ts">
// liunian 页免费流年岛（营销站）：出生信息 + 年份 → 流年目录盘。
// 盘面组件与查询生命周期均来自共享层，与 App 同源。
import { computed, reactive, ref } from 'vue'
import {
  baziLiunianFor,
  baziNatalFor,
  type BaziLiuNianOutput,
  type BaziNatal,
} from '@shared/domain/natal'
import type { BirthInput } from '@shared/domain/birth'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import YearField from '@shared/components/form/YearField.vue'
import { birthFormRules } from '@shared/components/form/birth-rules'
import { consultationBirthFromFields, type ConsultationContext } from '@shared/domain/consultation'
import { HANDOFF_KEYS, saveConsultationHandoff, saveHandoff } from '@shared/utils/handoff'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
    computing: string
    error: string
    result: string
    cta: string
    privacy: string
    }
}>()

const rules = birthFormRules(props.labels)

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const birthInput = ref<BirthInput | null>(null)
const status = ref<EngineResultStatus>('idle')
const bazi = ref<BaziNatal | null>(null)
const year = ref(new Date().getFullYear())

const loading = computed(() =>
  !!birthInput.value && (status.value === 'idle' || status.value === 'loading'),
)

function onBaziData(value: unknown) {
  bazi.value = (value as BaziNatal | null) ?? null
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  bazi.value = null
  birthInput.value = birthInputFromFields(birth)
}

function loadBazi(input: BirthInput) {
  return baziNatalFor(input)
}

function loadLiunian(input: { chart: BaziNatal['bazi']; year: number }): Promise<BaziLiuNianOutput> {
  return baziLiunianFor(input.chart, input.year)
}

function continueToApp() {
  if (!birthInput.value) return
  saveConsultationHandoff({
    version: 1,
    scenario: 'liunian',
    birth: consultationBirthFromFields(birth),
    year: Number(year.value),
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
    <BirthFields :value="birth" :labels="labels" @update:patch="Object.assign(birth, $event)">
      <YearField v-model="year" />
    </BirthFields>
  </FormShell>

  <EngineQueryPanel
    v-if="birthInput"
    resource="liunian.bazi"
    :input="birthInput"
    :load="loadBazi"
    :enabled="!!birthInput"
    :fallback="labels.error"
    @update:data="onBaziData"
    @update:status="onStatus"
  >
    <template #default="{ result }">
      <section class="tool-pan">
        <h3 class="tool-pan-title">
          {{ labels.result }}
        </h3>
        <BaziFullChartPanel
          v-if="result.baziDetail"
          :result="result.baziDetail"
          class="bazi-panel"
        >
          <template #before-yongshen>
            <EngineQueryPanel
              resource="liunian.bazi.liunian"
              :input="{ chart: result.bazi, year }"
              :load="loadLiunian"
              :enabled="!!result.bazi"
              :fallback="labels.error"
            >
              <template #default="{ result: annual }">
                <BaziLiuNianPanel
                  :result="annual"
                  class="bazi-panel"
                />
              </template>
            </EngineQueryPanel>
          </template>
        </BaziFullChartPanel>
        <template v-else>
          <BaziChartPanel
            :result="result.bazi"
            class="bazi-panel"
          />
          <EngineQueryPanel
            resource="liunian.bazi.liunian"
            :input="{ chart: result.bazi, year }"
            :load="loadLiunian"
            :enabled="!!result.bazi"
            :fallback="labels.error"
          >
            <template #default="{ result: annual }">
              <BaziLiuNianPanel
                :result="annual"
                class="bazi-panel"
              />
            </template>
          </EngineQueryPanel>
        </template>
      </section>
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
