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
import { toUserMessage } from '@/utils/errors'
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import {
  divinationCastingFor,
  divinationChartFor,
  type LiuyaoChartResult,
} from '@shared/domain/divination'
import { useToolRecords } from '@/composables/useToolRecords'
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
const snapshot = ref<LiuyaoChartResult | null>(null)
const lastCasting = ref<LiuyaoChartResult['casting'] | null>(null)
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
const loading = ref(false)
const error = ref('')

async function cast(context: DivinationCastingFormValue) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  const location = context.location
  if (!location) {
    loading.value = false
    return
  }
  const castingInstant = new Date().toISOString()
  snapshot.value = null
  try {
    const { casting } = await divinationCastingFor({ mode: context.castingMode })
    lastCasting.value = casting
    lastCastingInstant.value = castingInstant
    lastCastingLocation.value = location
    snapshot.value = await divinationChartFor({
      instant: castingInstant,
      timezone: location.timezone,
      longitude: location.longitude,
      casting,
      yongShen: '世爻',
    })
    // 卦不可复现：每次摇卦自动落一条历史记录。
    void saveToolRecord(
      recordTitle(context.question.trim()),
      {
        question: context.question.trim(),
        casting_mode: context.castingMode,
        casting_instant: castingInstant,
        location,
      },
      snapshot.value as unknown as Record<string, unknown>,
    )
  } catch (err) {
    error.value = toUserMessage(err, t('services.divination.failed'))
  } finally {
    loading.value = false
  }
}

function recordTitle(question: string): string {
  return t('records.titleLiuyao', { question: question.slice(0, 30) })
}

function openRecord(record: ToolRecord) {
  castingFormRef.value?.restore(record.input)
  snapshot.value = (record.output as unknown as LiuyaoChartResult | undefined) ?? null
  const input = record.input as {
    casting_instant?: string
    location?: DivinationCastingFormValue['location']
  }
  lastCasting.value = snapshot.value?.casting ?? null
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
