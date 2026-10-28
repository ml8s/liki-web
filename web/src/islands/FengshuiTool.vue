<script setup lang="ts">
// fengshui 页免费风水岛：出生年 + 性别 → 命卦/方位/流年飞星；另有玄空飞星盘与八宅布局。
// 三个结果块各自使用共享查询包装，任一失败不影响其他块。
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
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
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
const layout = reactive({ zuoShan: 0, periodDate: new Date().toISOString().slice(0, 10), doorGua: '', masterGua: '', stoveGua: '' })

const bazhaiInput = ref<{ birthYear: number; gender: string } | null>(null)
const bazhaiStatus = ref<EngineResultStatus>('idle')
const chart = ref<BazhaiChart | null>(null)
const layoutResult = ref<BazhaiLayout | null>(null)

const xuankongInput = computed(() => bazhaiInput.value
  ? { periodDate: layout.periodDate, zuoShan: layout.zuoShan }
  : null)
const layoutInput = computed(() => {
  const bazhai = chart.value
  if (!bazhai || !layout.doorGua || !layout.masterGua || !layout.stoveGua) return null
  return {
    mingGua: bazhai.ming_gua.gua.name,
    doorGua: layout.doorGua,
    masterGua: layout.masterGua,
    stoveGua: layout.stoveGua,
  }
})

const loading = computed(() =>
  !!bazhaiInput.value && (bazhaiStatus.value === 'idle' || bazhaiStatus.value === 'loading'),
)

function onBazhaiData(value: unknown) {
  chart.value = (value as BazhaiChart | null) ?? null
  if (!value) layoutResult.value = null
}

function onBazhaiStatus(value: EngineResultStatus) {
  bazhaiStatus.value = value
}

function onLayoutData(value: unknown) {
  layoutResult.value = (value as BazhaiLayout | null) ?? null
}

function loadBazhai(input: { birthYear: number; gender: string }) {
  return bazhaiChartFor(input.birthYear, input.gender)
}

function loadXuankong(input: { periodDate: string; zuoShan: number }): Promise<XuankongChart> {
  return xuankongChartFor(input.periodDate, input.zuoShan)
}

function loadLayout(input: { mingGua: string; doorGua: string; masterGua: string; stoveGua: string }) {
  return bazhaiLayoutFor(input.mingGua, input.doorGua, input.masterGua, input.stoveGua)
}

function patchLayout(patch: Partial<typeof layout>) {
  Object.assign(layout, patch)
}

function submit() {
  const value = Number(birth.year)
  if (!value || !birth.gender || loading.value) return
  chart.value = null
  layoutResult.value = null
  bazhaiInput.value = { birthYear: value, gender: birth.gender }
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

  <template v-if="bazhaiInput">
    <EngineQueryPanel
      resource="fengshui.bazhai"
      :input="bazhaiInput"
      :load="loadBazhai"
      :enabled="!!bazhaiInput"
      :fallback="labels.error"
      @update:data="onBazhaiData"
      @update:status="onBazhaiStatus"
    >
      <template #default="{ result }">
        <div class="tool-pan">
          <BazhaiChartPanel
            :result="result"
            :layout="layoutResult"
          />
        </div>
      </template>
    </EngineQueryPanel>

    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.xuankongTitle') }}
      </h3>
      <FengshuiLayoutFields :value="layout" @update:patch="patchLayout" />
      <EngineQueryPanel
        resource="fengshui.xuankong"
        :input="xuankongInput"
        :load="loadXuankong"
        :enabled="!!xuankongInput"
        :fallback="labels.error"
      >
        <template #default="{ result }">
          <XuankongChartPanel :result="result" />
        </template>
      </EngineQueryPanel>
    </div>

    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ t('services.fengshui.layoutTitle') }}
      </h3>
      <EngineQueryPanel
        resource="fengshui.layout"
        :input="layoutInput"
        :load="loadLayout"
        :enabled="!!layoutInput"
        :fallback="labels.error"
        @update:data="onLayoutData"
      >
        <template #default="{ result }">
          <BazhaiLayoutPanel :result="result" />
        </template>
      </EngineQueryPanel>
    </div>

    <ToolCta
      :href="appUrl"
      :privacy="labels.privacy"
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
    >
      {{ labels.cta }}
    </ToolCta>
  </template>
</template>
