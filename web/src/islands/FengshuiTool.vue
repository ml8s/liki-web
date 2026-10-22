<script setup lang="ts">
// fengshui 页免费风水岛：出生年 + 性别 → 命卦/方位/流年飞星；另有玄空飞星盘与八宅布局（门/主/灶）。
// 盘面组件来自共享层（与 App 的 FengshuiView 同源）。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  bazhaiChartFor,
  bazhaiLayoutFor,
  xuankongChartFor,
  type BazhaiChart,
  type BazhaiLayout,
  type XuankongChart,
} from '@shared/domain/fengshui'
import MingGuaSection from '@shared/components/charts/MingGuaSection.vue'
import LiuNianFlySection from '@shared/components/charts/LiuNianFlySection.vue'
import XuankongPanel from '@shared/components/charts/XuankongPanel.vue'
import BazhaiLayoutPanel from '@shared/components/charts/BazhaiLayoutPanel.vue'
import FengshuiBirthFields from '@shared/components/fengshui/FengshuiBirthFields.vue'
import FengshuiLayoutFields from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: {
    submit: string
    computing: string
    error: string
    cta: string
    privacy: string
  }
}>()

const { t } = useI18n()

const birth = reactive<{ year: number | null; gender: '' | 'female' | 'male' }>({ year: null, gender: '' })
const loading = ref(false)
const failed = ref(false)
const chart = ref<BazhaiChart | null>(null)

// 玄空飞星（坐山默认子山午向；向山 = 坐山 +12；立极日期默认今天）。
const layout = reactive({ zuoShan: 0, periodDate: new Date().toISOString().slice(0, 10), doorGua: '', masterGua: '', stoveGua: '' })
const xuankong = ref<XuankongChart | null>(null)

// 八宅布局（门/主/灶三卦；命卦来自上方结果）。layout 是唯一表单状态源，layoutResult 是接口结果。
const layoutResult = ref<BazhaiLayout | null>(null)
const layoutReady = computed(() => !!(chart.value && layout.doorGua && layout.masterGua && layout.stoveGua))

async function submit() {
  const value = Number(birth.year)
  if (!value || !birth.gender || loading.value) return
  loading.value = true
  failed.value = false
    chart.value = null
    layoutResult.value = null
  try {
    chart.value = await bazhaiChartFor(value, birth.gender)
    void runXuankong()
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

async function runXuankong() {
  try {
    xuankong.value = await xuankongChartFor(layout.periodDate, layout.zuoShan)
  } catch {
    xuankong.value = null
  }
}

async function runLayout() {
  const bazhai = chart.value
  if (!layoutReady.value || !bazhai) return
  try {
    layoutResult.value = await bazhaiLayoutFor(bazhai.ming_gua.gua.name, layout.doorGua, layout.masterGua, layout.stoveGua)
  } catch {
    layoutResult.value = null
  }
}

function patchLayout(patch: Partial<typeof layout>) {
  Object.assign(layout, patch)
  if (patch.zuoShan !== undefined || patch.periodDate !== undefined) void runXuankong()
  if (patch.doorGua !== undefined || patch.masterGua !== undefined || patch.stoveGua !== undefined) void runLayout()
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
    <FengshuiBirthFields :value="birth" @update:patch="Object.assign(birth, $event)" />
  </FormShell>

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="chart">
    <div class="tool-pan">
      <MingGuaSection :bazhai="chart" />
      <LiuNianFlySection
        v-if="chart.liu_nian_xing"
        :liu-nian="chart.liu_nian_xing"
      />
    </div>
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.xuankongTitle') }}
      </h3>
      <FengshuiLayoutFields :value="layout" @update:patch="patchLayout" />
      <XuankongPanel :chart="xuankong" />
    </div>
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.layoutTitle') }}
      </h3>
      <BazhaiLayoutPanel
        :layout="layoutResult"
        :bazhai="chart"
      />
    </div>
    <a class="btn-purple tool-cta" :href="appUrl">{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
