<template>
  <ServicePageShell
    :title="$t('services.divination.pageTitle')"
    :subtitle="$t('services.divination.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.divination.cast') }}
        </h2>
      </template>
      <DivinationCastingForm
        ref="castingFormRef"
        variant="full"
        :loading="loading"
        @submit="cast"
      />
    </el-card>

    <EngineQueryPanel
      v-if="divinationInput && !recordSnapshot"
      resource="divination.chart"
      :input="divinationInput"
      :load="loadDivinationChart"
      :enabled="!!divinationInput"
      :fallback="$t('services.divination.failed')"
    />

    <template v-if="snapshot">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <LiuyaoChartPanel :result="snapshot" />
      </el-card>

      <AppConsultationCard
        :title="$t('services.divination.consultationTitle')"
        :description="$t('services.divination.consultationBody')"
        :locked-description="$t('services.divination.lockedBody')"
        :purchased="entitlements.purchased && profiles.list.length > 0"
        :locked-action-label="profiles.list.length ? $t('commerce.activateNow') : $t('profiles.create')"
        :to="profiles.list.length ? '/chat' : '/birth-profiles/new'"
        @action="startChat"
      />
    </template>

    <el-card
      v-if="toolRecords.length || recordsError"
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('records.title') }}
        </h2>
      </template>
      <RecordList
        :records="toolRecords"
        :error="recordsError"
        @open="openRecord"
        @rename="renameToolRecord"
        @remove="removeToolRecord"
      />
    </el-card>
  </ServicePageShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import {
  divinationCastingFor,
  divinationChartFor,
  type DivinationChartInput,
  type LiuyaoChartResult,
} from '@shared/domain/divination'
import { useEngineResult } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import { useToolRecords } from '@/composables/useToolRecords'
import { useGuardedTask } from '@/composables/useGuardedTask'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'
import type { ConsultationContext } from '@shared/domain/consultation'
import { consumeConsultationHandoff } from '@shared/utils/handoff'
import LiuyaoChartPanel from '@shared/components/engine/liuyao/LiuyaoChartPanel.vue'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()
const {
  records: toolRecords,
  error: recordsError,
  load: loadRecords,
  save: saveToolRecord,
  confirmRename: renameToolRecord,
  confirmRemove: removeToolRecord,
} = useToolRecords('liuyao')
void loadRecords()
const publicConsultation = consumeConsultationHandoff<ConsultationContext>('divination')

const castingFormRef = ref<InstanceType<typeof DivinationCastingForm>>()
const recordSnapshot = ref<LiuyaoChartResult | null>(null)
const divinationInput = ref<DivinationChartInput | null>(null)
const chartResource = useEngineResult<DivinationChartInput, LiuyaoChartResult>({
  namespace: 'divination.chart',
  input: divinationInput,
  load: input => divinationChartFor(input),
})
const snapshot = computed(() => recordSnapshot.value ?? chartResource.data.value)
const lastCasting = ref<LiuyaoChartResult['casting'] | null>(null)
const lastQuestion = ref('')
const lastCastingMode = ref<DivinationCastingFormValue['castingMode']>('auto')
let pendingRecordSave = false
const lastCastingInstant = ref('')
const lastCastingLocation = ref<DivinationCastingFormValue['location']>(null)
if (publicConsultation?.scenario === 'divination') {
  lastCasting.value = publicConsultation.casting ?? null
  lastCastingInstant.value = publicConsultation.casting_instant ?? ''
  lastCastingLocation.value = publicConsultation.location
    ? {
        place: '',
        longitude: publicConsultation.location.longitude,
        timezone: publicConsultation.location.timezone,
        source: 'saved',
      }
    : null
}
const error = ref('')
const castTask = useGuardedTask({ report: message => { error.value = message } })
const loading = computed(() =>
  castTask.loading.value || chartResource.status.value === 'loading',
)

function loadDivinationChart(input: DivinationChartInput) {
  return divinationChartFor(input)
}

function cast(context: DivinationCastingFormValue) {
  return castTask.run(async () => {
    error.value = ''
    const location = context.location
    if (!location) return
    const castingInstant = new Date().toISOString()
    recordSnapshot.value = null
    const { casting } = await divinationCastingFor({ mode: context.castingMode })
    lastCasting.value = casting
    lastCastingInstant.value = castingInstant
    lastCastingLocation.value = location
    lastQuestion.value = context.question.trim()
    lastCastingMode.value = context.castingMode
    pendingRecordSave = true
    divinationInput.value = {
      instant: castingInstant,
      timezone: location.timezone,
      longitude: location.longitude,
      casting,
      yongShen: '世爻',
    }
  }, t('services.divination.failed'))
}

// 卦不可复现：每次摇卦在图表查询成功后自动落一条历史记录。
watch(() => chartResource.data.value, chart => {
  if (!chart || !pendingRecordSave) return
  pendingRecordSave = false
  void saveToolRecord(
    recordTitle(lastQuestion.value),
    {
      question: lastQuestion.value,
      casting_mode: lastCastingMode.value,
      casting_instant: lastCastingInstant.value,
      location: lastCastingLocation.value,
    },
    chart as unknown as Record<string, unknown>,
  )
})

function recordTitle(question: string): string {
  return t('records.titleLiuyao', { question: question.slice(0, 30) })
}

function openRecord(record: ToolRecord) {
  castingFormRef.value?.restore(record.input)
  chartResource.reset()
  divinationInput.value = null
  recordSnapshot.value = (record.output as unknown as LiuyaoChartResult | undefined) ?? null
  const input = record.input as {
    casting_instant?: string
    location?: DivinationCastingFormValue['location']
  }
  lastCasting.value = recordSnapshot.value?.casting ?? null
  lastCastingInstant.value = input.casting_instant ?? ''
  lastCastingLocation.value = input.location ?? null
}

onMounted(() => {
  if (publicConsultation?.scenario !== 'divination') return
  castingFormRef.value?.restore({
    question: publicConsultation.question ?? '',
    location: lastCastingLocation.value,
  })
})

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  const question = castingFormRef.value?.getContext().question.trim() ?? ''
  chat.pendingIntent = t('services.divination.intent', { question })
  if (!lastCasting.value || !lastCastingInstant.value || !lastCastingLocation.value) return
  const consultationContext: ConsultationContext = {
    version: 1,
    scenario: 'divination',
    question,
    casting: lastCasting.value,
    casting_instant: lastCastingInstant.value,
    location: {
      longitude: lastCastingLocation.value.longitude,
      timezone: lastCastingLocation.value.timezone,
    },
  }
  void openServiceChat(profile.id, 'divination', message => { error.value = message }, consultationContext)
}
</script>
