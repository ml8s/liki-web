<script setup lang="ts">
// natal 页免费排盘岛（营销站）：出生信息 → 共享排盘链路 → 八字盘 + 紫微盘。
// 盘面组件来自共享层（web/shared/components/charts），与 App 使用同一实现；
// 组件文案（profiles.chart.*）由挂载脚本安装的 vue-i18n 提供。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { natalChartFor, type NatalChart } from '@shared/domain/natal'
import { baziLiunianFor, type BaziLiuNianOutput } from '@shared/domain/natal'
import { WUXING_LUCKY } from '@shared/utils/wuxing'
import BaziPanel from '@shared/components/charts/BaziPanel.vue'
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
import ZiweiPanel from '@shared/components/charts/ZiweiPanel.vue'
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
    lucky: string
    cta: string
    privacy: string
    }
}>()

const { t } = useI18n()

const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const loading = ref(false)
const failed = ref(false)
const natal = ref<NatalChart | null>(null)
const liunian = ref<BaziLiuNianOutput | null>(null)
const liunianYear = ref(new Date().getFullYear())

// 流年盘（bazi.liunian）：随八字盘加载；失败不阻塞主盘。
async function loadLiunian() {
  const chart = natal.value?.bazi
  if (!chart) return
  try {
    liunian.value = await baziLiunianFor(chart, Number(liunianYear.value))
  } catch {
    liunian.value = null
  }
}

// 幸运元素（五行统计/状态由共享 BaziPanel 用引擎数据展示）。
const luckyElements = computed(() => {
  // 幸运元素只消费确定性候选：扶抑 yong/xi，调候 primary/secondary。null 表示无候选。
  const full = natal.value?.baziDetail
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

async function submit() {
  if (!isBirthInputReady(birth) || loading.value) return
  loading.value = true
  failed.value = false
  liunian.value = null
  try {
    natal.value = await natalChartFor(birthInputFromFields(birth))
    await loadLiunian()
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function continueToApp() {
  if (!natal.value) return
  saveConsultationHandoff({
    version: 1,
    scenario: 'natal',
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
    <BirthFields :value="birth" :labels="labels" @update:patch="Object.assign(birth, $event)" />
  </FormShell>

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="natal">
    <section class="tool-pan">
      <h3 class="tool-pan-title">
        {{ labels.bazi }}
      </h3>
      <YearField
        v-model="liunianYear"
        name="natal-liunian-year"
        @update:model-value="loadLiunian"
      />
      <BaziPanel
        :chart="natal.bazi"
        :full="natal.baziDetail"
        :liunian="liunian"
      />
      <div v-if="luckyElements" class="wuxing-row">
        <span class="tool-pan-sub">{{ labels.lucky }}</span>
        <span v-for="l in luckyElements" :key="l.element" class="lucky-chip">
          <span class="lucky-dots">
            <i v-for="c in l.colors" :key="c" class="lucky-dot" :style="{ background: c }" />
          </span>
          {{ l.element }} · {{ l.numbers }}
        </span>
      </div>
    </section>
    <section
      v-if="natal.ziwei"
      class="tool-pan"
    >
      <h3 class="tool-pan-title">
        {{ labels.ziwei }}
      </h3>
      <ZiweiPanel
        :chart="natal.ziwei"
        :da-xian="natal.ziweiDaXian"
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
.wuxing-row { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-2); margin-top: var(--space-3); }
.tool-pan-sub { font-size: var(--font-size-label); color: var(--color-text-muted); }
.lucky-chip { display: inline-flex; align-items: center; gap: var(--space-2); padding: 2px 10px; border-radius: var(--radius-full); border: 1px solid var(--color-border); font-size: var(--font-size-caption); }
.lucky-dots { display: inline-flex; gap: 3px; }
.lucky-dot { display: inline-block; width: 10px; height: 10px; border-radius: var(--radius-full); border: 1px solid var(--color-border); }
</style>
