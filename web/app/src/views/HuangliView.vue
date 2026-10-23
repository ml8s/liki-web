<template>
  <ServicePageShell
    :title="$t('services.huangli.pageTitle')"
    :subtitle="$t('services.huangli.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.huangli.formTitle') }}
        </h2>
      </template>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent="calc"
      >
        <HuangliRangeFields
          :value="form"
          @update:patch="patchRange"
        />
        <el-button
          type="primary"
          native-type="submit"
          :loading="loading"
        >
          {{ $t('services.huangli.cta') }}
        </el-button>
        <el-button
          v-if="result?.length"
          @click="saveRecord"
        >
          {{ $t('records.save') }}
        </el-button>
      </el-form>
    </el-card>

    <template v-if="result">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.huangli.resultTitle', { event: eventLabel(event) }) }}
          </h2>
        </template>
        <HuangliDaysPanel :days="result" />
      </el-card>

      <AppConsultationCard
        :title="$t('services.huangli.consultationTitle')"
        :description="$t('services.huangli.consultationBody')"
        :locked-description="$t('services.huangli.lockedBody')"
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
import { computed, reactive, ref, toRef, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { huangliDaysFor, type HuangliDays } from '@shared/domain/huangli'
import HuangliDaysPanel from '@shared/components/charts/HuangliDaysPanel.vue'
import HuangliRangeFields from '@shared/components/huangli/HuangliRangeFields.vue'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import type { ConsultationContext } from '@shared/domain/consultation'
import { consumeConsultationHandoff } from '@shared/utils/handoff'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()
const {
  records: toolRecords,
  error: recordsError,
  load: loadRecords,
  save: saveToolRecord,
  confirmRename: renameToolRecord,
  confirmRemove: removeToolRecord,
} = useToolRecords('huangli')
void loadRecords()

// 表单模型保持稳定对象，v-model 写入同一引用，change 当拍校验才能读到新值。
const form = reactive({ event: 'wedding', startDate: '', endDate: '' })
const publicConsultation = consumeConsultationHandoff<ConsultationContext>('huangli')
if (publicConsultation?.scenario === 'huangli') {
  form.event = publicConsultation.event ?? form.event
  form.startDate = publicConsultation.start_date ?? ''
  form.endDate = publicConsultation.end_date ?? ''
}
const event = toRef(form, 'event')
const startDate = toRef(form, 'startDate')
const endDate = toRef(form, 'endDate')
const result = ref<HuangliDays | null>(null)
const loading = ref(false)
const error = ref('')

const formRef = ref<FormInstance>()

function patchRange(patch: Partial<typeof form>) {
  Object.assign(form, patch)
}
const spanDays = computed(() => {
  if (!startDate.value || !endDate.value) return 0
  return Math.round((new Date(endDate.value).getTime() - new Date(startDate.value).getTime()) / 86400000) + 1
})
function validateRange(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value || !startDate.value) return callback()
  if (startDate.value > value) return callback(new Error(t('services.huangli.endBeforeStart')))
  if (spanDays.value > 30) return callback(new Error(t('services.huangli.rangeTooLong', { days: spanDays.value })))
  callback()
}
const rules = computed<FormRules>(() => ({
  startDate: [{ required: true, message: t('profiles.form.selectDate'), trigger: 'change' }],
  endDate: [
    { required: true, message: t('profiles.form.selectDate'), trigger: 'change' },
    { validator: validateRange, trigger: 'change' },
  ],
}))
// 起始日期变化后，已填写的结束日期需要重新校验范围。
watch(startDate, () => {
  if (endDate.value) void formRef.value?.validateField('endDate').catch(() => {})
})

function eventLabel(key: string): string {
  return t(`services.huangli.events.${key}`)
}

// 引擎按 start_date + count 连续取数（count ≤ 30），结束日期只用于算跨度。
async function calc() {
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  loading.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await huangliDaysFor(event.value, startDate.value, spanDays.value)
  } catch (err) {
    error.value = toUserMessage(err, t('services.huangli.failed'))
  } finally {
    loading.value = false
  }
}

function saveRecord() {
  if (!result.value?.length) return
  void saveToolRecord(
    t('records.titleHuangli', { event: eventLabel(event.value), start: startDate.value }),
    { event: event.value, start_date: startDate.value, end_date: endDate.value },
  )
}

function openRecord(record: ToolRecord) {
  event.value = String(record.input.event ?? 'wedding')
  startDate.value = String(record.input.start_date ?? '')
  endDate.value = String(record.input.end_date ?? '')
  void calc()
}

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  chat.pendingIntent = t('services.huangli.intent', { event: eventLabel(event.value) })
  // 择日属占卜域：场景走 huangli（标题/意图），内部路由到 divination 专家部署。
  const context: ConsultationContext = {
    version: 1,
    scenario: 'huangli',
    event: event.value,
    start_date: startDate.value,
    end_date: endDate.value,
  }
  void openServiceChat(profile.id, 'huangli', message => { error.value = message }, context)
}

</script>

<style scoped>
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: var(--space-3); }
</style>
