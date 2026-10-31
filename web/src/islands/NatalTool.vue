<script setup lang="ts">
// natal 页免费排盘岛（营销站）：出生信息 → 共享资源包装 → 八字目录盘 + 紫微目录盘。
// 盘面组件与查询生命周期（loading/error/retry）均来自共享层，与 App 使用同一实现。
import { computed, reactive, ref } from 'vue'
import {
  baziLiunianFor,
  baziNatalFor,
  ziweiLiunianFor,
  ziweiNatalFor,
  type BaziLiuNianOutput,
  type BaziNatal,
  type ZiweiNatal,
  type ZiweiChart,
  type ZiweiLiuNianData,
} from '@shared/domain/natal'
import type { BirthInput } from '@shared/domain/birth'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import { WUXING_LUCKY } from '@shared/utils/wuxing'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import YearField from '@shared/components/form/YearField.vue'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import ZiweiDaXianPanel from '@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'
import ZiweiLiuNianPanel from '@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'
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
    bazi: string
    ziwei: string
    baziIntro: string
    ziweiIntro: string
    ziweiAnnualIntro: string
    lucky: string
    cta: string
    privacy: string
    }
}>()

const rules = birthFormRules(props.labels)

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const birthInput = ref<BirthInput | null>(null)
const baziStatus = ref<EngineResultStatus>('idle')
const ziweiStatus = ref<EngineResultStatus>('idle')
const bazi = ref<BaziNatal | null>(null)
const ziwei = ref<ZiweiNatal | null>(null)
const liunianYear = ref(new Date().getFullYear())
const ziweiLiuNianYear = ref(new Date().getFullYear())

const loading = computed(() =>
  !!birthInput.value
  && (
    baziStatus.value === 'idle' || baziStatus.value === 'loading'
    || ziweiStatus.value === 'idle' || ziweiStatus.value === 'loading'
  ),
)

const luckyElements = computed(() => {
  const full = bazi.value?.baziDetail
  if (!full) return null
  const raw = [
    full.fu_yi?.xi,
    full.fu_yi?.yong,
    full.tiao_hou?.primary_wuxing,
    full.tiao_hou?.secondary_wuxing,
  ].map(value => value ?? '').join('')
  const elements = [...new Set(Array.from(raw).filter(w => w in WUXING_LUCKY))]
  if (!elements.length) return null
  return elements.map(w => ({ element: w, ...WUXING_LUCKY[w] }))
})

function onBaziData(value: unknown) {
  bazi.value = (value as BaziNatal | null) ?? null
}

function onZiweiData(value: unknown) {
  ziwei.value = (value as ZiweiNatal | null) ?? null
}

function onBaziStatus(status: EngineResultStatus) {
  baziStatus.value = status
}

function onZiweiStatus(status: EngineResultStatus) {
  ziweiStatus.value = status
}

function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  bazi.value = null
  ziwei.value = null
  birthInput.value = birthInputFromFields(birth)
}

function loadBazi(input: BirthInput) {
  return baziNatalFor(input)
}

function loadZiwei(input: BirthInput) {
  return ziweiNatalFor(input)
}

function loadLiunian(input: { chart: BaziNatal['bazi']; year: number }): Promise<BaziLiuNianOutput> {
  return baziLiunianFor(input.chart, input.year)
}

function loadZiweiLiunian(input: { base: ZiweiChart; year: number }): Promise<ZiweiLiuNianData> {
  return ziweiLiunianFor(input.base, input.year)
}

function continueToApp() {
  if (!birthInput.value) return
  saveConsultationHandoff({
    version: 1,
    intent: 'natal',
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
    <BirthFields :value="birth" :labels="labels" @update:patch="Object.assign(birth, $event)" />
  </FormShell>

  <EngineQueryPanel
    v-if="birthInput"
    resource="natal.bazi"
    :input="birthInput"
    :load="loadBazi"
    :enabled="!!birthInput"
    :fallback="labels.error"
    @update:data="onBaziData"
    @update:status="onBaziStatus"
  >
    <template #default="{ result }">
      <section class="tool-pan">
        <h3 class="tool-pan-title">
          {{ labels.bazi }}
        </h3>
        <p class="tool-pan-intro">
          {{ labels.baziIntro }}
        </p>
        <YearField
          v-model="liunianYear"
          name="natal-liunian-year"
        />
        <BaziFullChartPanel
          v-if="result.baziDetail"
          :result="result.baziDetail"
          class="bazi-panel"
        >
          <template #before-yongshen>
            <EngineQueryPanel
              resource="natal.bazi.liunian"
              :input="{ chart: result.bazi, year: liunianYear }"
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
            resource="natal.bazi.liunian"
            :input="{ chart: result.bazi, year: liunianYear }"
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
        <div
          v-if="luckyElements"
          class="wuxing-row"
        >
          <span class="tool-pan-sub">{{ labels.lucky }}</span>
          <span
            v-for="l in luckyElements"
            :key="l.element"
            class="lucky-chip"
          >
            <span class="lucky-dots">
              <i
                v-for="c in l.colors"
                :key="c"
                class="lucky-dot"
                :style="{ background: c }"
              />
            </span>
            {{ l.element }} · {{ l.numbers }}
          </span>
        </div>
      </section>
    </template>
  </EngineQueryPanel>

  <EngineQueryPanel
    v-if="birthInput"
    resource="natal.ziwei"
    :input="birthInput"
    :load="loadZiwei"
    :enabled="!!birthInput"
    :fallback="labels.error"
    @update:data="onZiweiData"
    @update:status="onZiweiStatus"
  >
    <template #default="{ result }">
      <section
        v-if="result.ziwei"
        class="tool-pan"
      >
        <h3 class="tool-pan-title">
          {{ labels.ziwei }}
        </h3>
        <p class="tool-pan-intro">
          {{ labels.ziweiIntro }}
        </p>
        <YearField
          v-model="ziweiLiuNianYear"
          name="natal-ziwei-liunian-year"
        />
        <ZiweiChartPanel
          :result="result.ziwei"
          class="ziwei-panel"
        />
        <ZiweiDaXianPanel
          v-if="result.ziweiDaXian.length"
          :result="result.ziweiDaXian"
          class="ziwei-panel"
        />
        <p class="tool-pan-intro">
          {{ labels.ziweiAnnualIntro }}
        </p>
        <EngineQueryPanel
          resource="natal.ziwei.liunian"
          :input="{ base: result.ziweiBase, year: ziweiLiuNianYear }"
          :load="loadZiweiLiunian"
          :enabled="!!result.ziweiBase"
          :fallback="labels.error"
        >
          <template #default="{ result: annual }">
            <ZiweiLiuNianPanel
              :result="annual"
              class="ziwei-panel"
            />
          </template>
        </EngineQueryPanel>
      </section>
    </template>
  </EngineQueryPanel>

  <ToolCta
    v-if="birthInput"
    :href="appUrl"
    :privacy="labels.privacy"
    @click="continueToApp"
  >
    {{ labels.cta }}
  </ToolCta>
</template>


<style scoped>
.wuxing-row { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-2); margin-top: var(--space-3); }
.tool-pan-sub { font-size: var(--font-size-label); color: var(--color-text-muted); }
.lucky-chip { display: inline-flex; align-items: center; gap: var(--space-2); padding: 2px 10px; border-radius: var(--radius-full); border: 1px solid var(--color-border); font-size: var(--font-size-caption); }
.lucky-dots { display: inline-flex; gap: 3px; }
.lucky-dot { display: inline-block; width: 10px; height: 10px; border-radius: var(--radius-full); border: 1px solid var(--color-border); }
</style>
