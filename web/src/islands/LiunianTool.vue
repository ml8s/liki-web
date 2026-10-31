<script setup lang="ts">
// liunian 页免费流年岛：出生信息 + 起始年份 → 八字十年 + 紫微十年。
// 概览与选中年份详情复用共享确定性面板；缺失年份显式呈现，不前端伪造。
import { computed, reactive, ref, watch } from 'vue'
import {
  baziLiunianRangeFor,
  baziNatalFor,
  LIUNIAN_DECADE_SIZE,
  mergeAnnualDecade,
  ziweiLiunianRangeFor,
  ziweiNatalFor,
  type AnnualDecadeItem,
  type BaziLiuNianOutput,
  type BaziNatal,
  type ZiweiChart,
  type ZiweiLiuNianData,
  type ZiweiNatal,
} from '@shared/domain/natal'
import type { BirthInput } from '@shared/domain/birth'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import ZiweiLiuNianPanel from '@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'
import AnnualDecadeOverview from '@shared/components/annual/AnnualDecadeOverview.vue'
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
    baziIntro: string
    baziAnnualIntro: string
    ziweiIntro: string
    ziweiAnnualIntro: string
    decadeTitle: string
    decadeIntro: string
    cta: string
    privacy: string
  }
}>()

const rules = birthFormRules(props.labels)

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const birthInput = ref<BirthInput | null>(null)
const status = ref<EngineResultStatus>('idle')
const bazi = ref<BaziNatal | null>(null)
const ziweiStatus = ref<EngineResultStatus>('idle')
const ziwei = ref<ZiweiNatal | null>(null)
const startYear = ref(new Date().getFullYear())
const selectedYear = ref(new Date().getFullYear())
const baziAnnualYears = ref<BaziLiuNianOutput[]>([])
const ziweiAnnualYears = ref<ZiweiLiuNianData[]>([])

const loading = computed(() =>
  !!birthInput.value
  && (
    status.value === 'idle' || status.value === 'loading'
    || ziweiStatus.value === 'idle' || ziweiStatus.value === 'loading'
  ),
)

function onBaziData(value: unknown) {
  bazi.value = (value as BaziNatal | null) ?? null
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function onZiweiData(value: unknown) {
  ziwei.value = (value as ZiweiNatal | null) ?? null
}

function onZiweiStatus(value: EngineResultStatus) {
  ziweiStatus.value = value
}

function onBaziAnnualData(value: unknown) {
  baziAnnualYears.value = Array.isArray(value)
    ? value as BaziLiuNianOutput[]
    : []
}

function onZiweiAnnualData(value: unknown) {
  ziweiAnnualYears.value = Array.isArray(value)
    ? value as ZiweiLiuNianData[]
    : []
}

function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  bazi.value = null
  ziwei.value = null
  baziAnnualYears.value = []
  ziweiAnnualYears.value = []
  selectedYear.value = startYear.value
  birthInput.value = birthInputFromFields(birth)
}

function loadBazi(input: BirthInput) {
  return baziNatalFor(input)
}

function loadBaziLiunianRange(input: {
  chart: BaziNatal['bazi']
  startYear: number
  count: number
}): Promise<BaziLiuNianOutput[]> {
  return baziLiunianRangeFor(input.chart, input.startYear, input.count)
}

function loadZiwei(input: BirthInput) {
  return ziweiNatalFor(input)
}

function loadZiweiLiunianRange(input: {
  base: ZiweiChart
  startYear: number
  count: number
}): Promise<ZiweiLiuNianData[]> {
  return ziweiLiunianRangeFor(input.base, input.startYear, input.count)
}

const decadeItems = computed(() =>
  mergeAnnualDecade(
    startYear.value,
    baziAnnualYears.value,
    ziweiAnnualYears.value,
    LIUNIAN_DECADE_SIZE,
  ),
)

const selectedAnnualItem = computed<AnnualDecadeItem | null>(() =>
  decadeItems.value.find(item => item.year === selectedYear.value) ?? null,
)

watch([baziAnnualYears, ziweiAnnualYears, startYear], () => {
  if (!decadeItems.value.some(item => item.year === selectedYear.value)) {
    selectedYear.value = startYear.value
  }
})

function continueToApp() {
  if (!birthInput.value) return
  saveConsultationHandoff({
    version: 1,
    intent: 'liunian',
    birth: consultationBirthFromFields(birth),
    year: selectedYear.value,
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
      <YearField
        v-model="startYear"
        name="liunian-start-year"
      />
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
      <details class="tool-pan">
        <summary class="tool-pan-title">
          {{ labels.result }}
        </summary>
        <p class="tool-pan-intro">
          {{ labels.baziIntro }}
        </p>
        <BaziFullChartPanel
          v-if="result.baziDetail"
          :result="result.baziDetail"
          class="bazi-panel"
        />
        <BaziChartPanel
          v-else
          :result="result.bazi"
          class="bazi-panel"
        />
        <EngineQueryPanel
          resource="liunian.bazi.decade"
          :input="{ chart: result.bazi, startYear, count: LIUNIAN_DECADE_SIZE }"
          :load="loadBaziLiunianRange"
          :enabled="!!result.bazi"
          :fallback="labels.error"
          @update:data="onBaziAnnualData"
        >
          <template #default="{ result: annualYears }">
            <span
              class="flow-visually-hidden"
              aria-live="polite"
            >{{ $t('profiles.chart.annualDecade') }} {{ annualYears.length }}</span>
          </template>
        </EngineQueryPanel>
      </details>
    </template>
  </EngineQueryPanel>

  <EngineQueryPanel
    v-if="birthInput"
    resource="liunian.ziwei"
    :input="birthInput"
    :load="loadZiwei"
    :enabled="!!birthInput"
    :fallback="labels.error"
    @update:data="onZiweiData"
    @update:status="onZiweiStatus"
  >
    <template #default="{ result }">
      <details
        v-if="result.ziwei"
        class="tool-pan"
      >
        <summary class="tool-pan-title">
          {{ $t('profiles.chart.stage.ziwei_liunian') }}
        </summary>
        <p class="tool-pan-intro">
          {{ labels.ziweiIntro }}
        </p>
        <ZiweiChartPanel
          :result="result.ziwei"
          class="ziwei-panel"
        />
        <EngineQueryPanel
          resource="liunian.ziwei.decade"
          :input="{ base: result.ziweiBase, startYear, count: LIUNIAN_DECADE_SIZE }"
          :load="loadZiweiLiunianRange"
          :enabled="!!result.ziweiBase"
          :fallback="labels.error"
          @update:data="onZiweiAnnualData"
        >
          <template #default="{ result: annualYears }">
            <span
              class="flow-visually-hidden"
              aria-live="polite"
            >{{ $t('profiles.chart.annualDecade') }} {{ annualYears.length }}</span>
          </template>
        </EngineQueryPanel>
      </details>
    </template>
  </EngineQueryPanel>

  <section
    v-if="decadeItems.length"
    class="tool-pan"
  >
    <h3 class="tool-pan-title">
      {{ labels.decadeTitle }}
    </h3>
    <p class="tool-pan-intro">
      {{ labels.decadeIntro }}
    </p>
    <AnnualDecadeOverview
      :years="decadeItems"
      :selected-year="selectedYear"
      @select="selectedYear = $event"
    />

    <div
      v-if="selectedAnnualItem"
      class="annual-detail"
    >
      <BaziLiuNianPanel
        v-if="selectedAnnualItem.bazi"
        :result="selectedAnnualItem.bazi"
        class="bazi-panel"
      />
      <ZiweiLiuNianPanel
        v-if="selectedAnnualItem.ziwei"
        :result="selectedAnnualItem.ziwei"
        class="ziwei-panel"
      />
      <p
        v-if="!selectedAnnualItem.bazi || !selectedAnnualItem.ziwei"
        class="annual-detail__partial"
      >
        {{ $t('profiles.chart.annualPartial') }}
      </p>
    </div>
  </section>

  <ToolCta
    :href="appUrl"
    :privacy="labels.privacy"
    @click="continueToApp"
  >
    {{ labels.cta }}
  </ToolCta>
</template>

<style scoped>
.annual-detail {
  margin-top: var(--space-4);
  display: grid;
  gap: var(--space-4);
  min-width: 0;
}

.annual-detail__partial {
  margin: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}

.tool-pan > summary {
  cursor: pointer;
}
</style>
