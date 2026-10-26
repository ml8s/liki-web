<script setup lang="ts">
// fengshui 页免费风水岛：出生年 + 性别 → 命卦/方位/流年飞星；另有玄空飞星盘与八宅布局（门/主/灶）。
// 盘面使用 engine result catalog 的三个标准 result family。
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
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import BazhaiLayoutPanel from '@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'
import XuankongChartPanel from '@shared/components/engine/fengshui/XuankongChartPanel.vue'
import FengshuiBirthFields from '@shared/components/fengshui/FengshuiBirthFields.vue'
import FengshuiLayoutFields from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import { saveConsultationHandoff } from '@shared/utils/handoff'

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
      <BazhaiChartPanel
        :result="chart"
        :layout="layoutResult"
      />
    </div>
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.xuankongTitle') }}
      </h3>
      <FengshuiLayoutFields :value="layout" @update:patch="patchLayout" />
      <XuankongChartPanel
        v-if="xuankong"
        :result="xuankong"
      />
    </div>
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.layoutTitle') }}
      </h3>
      <BazhaiLayoutPanel
        v-if="layoutResult"
        :result="layoutResult"
      />
    </div>
    <a
      class="btn-purple tool-cta"
      :href="appUrl"
      @click="saveConsultationHandoff({
        version: 1,
        scenario: 'fengshui',
        birth_year: birth.year ?? undefined,
        gender: birth.gender || undefined,
        layout: {
          zuo_shan: layout.zuoShan,
          period_date: layout.periodDate,
          door_gua: layout.doorGua,
          master_gua: layout.masterGua,
          stove_gua: layout.stoveGua,
        },
      })"
    >{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
