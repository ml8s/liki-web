<script setup lang="ts">
// divination 页免费起卦岛：所问 → 起卦 + 确定性六爻排盘。
// 公开层展示完整卦象、六亲、纳甲、格局与应期等专业事实；AI 解读仍在 App。
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  divinationCastingFor,
  divinationChartFor,
  type LiuyaoCasting,
} from '@shared/domain/divination'
import LiuyaoCastingPanel from '@shared/components/engine/liuyao/LiuyaoCastingPanel.vue'
import LiuyaoChartPanel from '@shared/components/engine/liuyao/LiuyaoChartPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import ToolError from '@shared/components/tools/ToolError.vue'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'
import type { DivinationChartInput } from '@shared/domain/divination'
import type { ConsultationContext } from '@shared/domain/consultation'
import { standardTimeFromInstant } from '@shared/domain/time'
import { saveConsultationHandoff } from '@shared/utils/handoff'

const props = defineProps<{
  appUrl: string
  labels: {
    error: string
    castingIntro: string
    chartIntro: string
    cta: string
    privacy: string
  }
}>()

const { t } = useI18n()

const question = ref('')
const castingInstant = ref('')
const castingLocation = ref<DivinationCastingFormValue['location']>(null)
const loading = ref(false)
const failed = ref(false)
const casting = ref<LiuyaoCasting | null>(null)
const chartInput = ref<DivinationChartInput | null>(null)

async function submit(value: DivinationCastingFormValue) {
  if (loading.value) return
  loading.value = true
  failed.value = false
  casting.value = null
  chartInput.value = null
  question.value = value.question.trim()
  try {
    const submittedAt = new Date().toISOString()
    const result = await divinationCastingFor({ mode: value.castingMode })
    casting.value = result.casting
    castingInstant.value = submittedAt
    castingLocation.value = value.location
    chartInput.value = value.location
      ? {
          standard_time: standardTimeFromInstant(submittedAt, value.location.timezone),
          longitude: value.location.longitude,
          casting: result.casting,
          yongShen: '世爻',
        }
      : null
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <DivinationCastingForm
    variant="full"
    :loading="loading"
    @submit="submit"
  />

  <ToolError :message="failed ? labels.error : ''" />

  <template v-if="casting">
    <div class="tool-pan">
      <h3 class="tool-pan-title">
        {{ question ? t('services.divination.questionLabel') + '：' + question : t('services.divination.cast') }}
      </h3>
      <p class="tool-pan-intro">
        {{ labels.castingIntro }}
      </p>
      <LiuyaoCastingPanel :result="casting" />
    </div>
    <EngineQueryPanel
      v-if="chartInput"
      resource="divination.chart"
      :input="chartInput"
      :load="divinationChartFor"
      :enabled="!!chartInput"
      :fallback="labels.error"
    >
      <template #default="{ result }">
        <div class="tool-pan">
          <h3 class="tool-pan-title">
            {{ t('services.divination.hexagram') }}
          </h3>
          <p class="tool-pan-intro">
            {{ labels.chartIntro }}
          </p>
          <LiuyaoChartPanel :result="result" />
        </div>
      </template>
    </EngineQueryPanel>
    <ToolCta
      :href="appUrl"
      :privacy="labels.privacy"
      @click="saveConsultationHandoff({
        version: 1,
        scenario: 'divination',
        question: question,
        casting,
        casting_instant: castingInstant,
        location: castingLocation && {
          longitude: castingLocation.longitude,
          timezone: castingLocation.timezone,
        },
      } as ConsultationContext)"
    >
      {{ labels.cta }}
    </ToolCta>
  </template>
</template>
