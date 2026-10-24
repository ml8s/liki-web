<script setup lang="ts">
// liunian 页免费流年岛（营销站）：出生信息 + 年份 → 流年盘。
// 盘面组件来自共享层（BaziPanel 的流年列与流年详览），与 App 同源。
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { baziFullChartFor, type BaziFullChart } from '@shared/domain/natal'
import { baziLiunianFor, type BaziLiuNianOutput } from '@shared/domain/natal'
import BaziPanel from '@shared/components/charts/BaziPanel.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import YearField from '@shared/components/form/YearField.vue'
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

const { t } = useI18n()

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const year = ref(new Date().getFullYear())
const loading = ref(false)
const failed = ref(false)
const pan = ref<BaziFullChart | null>(null)
const liunian = ref<BaziLiuNianOutput | null>(null)

// 流年盘（bazi.liunian）：随年份切换刷新；失败不阻塞主盘。
async function loadLiunian() {
  const current = pan.value
  if (!current) return
  try {
    liunian.value = await baziLiunianFor(current.chart, Number(year.value))
  } catch {
    liunian.value = null
  }
}

async function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  loading.value = true
  failed.value = false
  pan.value = null
  liunian.value = null
  try {
    pan.value = await baziFullChartFor(birthInputFromFields(birth))
    await loadLiunian()
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function continueToApp() {
  if (!pan.value) return
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
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <BirthFields :value="birth" :labels="labels" @update:patch="Object.assign(birth, $event)">
      <YearField
        v-model="year"
        @update:model-value="loadLiunian"
      />
    </BirthFields>
  </FormShell>

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="pan">
    <section class="tool-pan">
      <h3 class="tool-pan-title">
        {{ labels.result }}
      </h3>
      <BaziPanel
        :chart="pan.chart"
        :full="pan.full"
        :liunian="liunian"
      />
    </section>
    <a
      class="btn-purple tool-cta"
      :href="appUrl"
      @click="continueToApp"
    >{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>

<style scoped>
</style>
