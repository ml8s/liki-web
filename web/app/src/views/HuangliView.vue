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
        <div class="row">
          <el-form-item :label="$t('services.huangli.event')">
            <el-select
              v-model="event"
              :aria-label="$t('services.huangli.event')"
              class="w-full"
            >
              <el-option
                v-for="key in EVENT_KEYS"
                :key="key"
                :value="key"
                :label="$t(`services.huangli.events.${key}`)"
              />
            </el-select>
          </el-form-item>
          <el-form-item
            prop="startDate"
            :label="$t('services.huangli.startDate')"
          >
            <el-date-picker
              v-model="startDate"
              type="date"
              value-format="YYYY-MM-DD"
              :placeholder="$t('services.huangli.startDate')"
              class="w-full"
            />
          </el-form-item>
          <el-form-item
            prop="endDate"
            :label="$t('services.huangli.endDate')"
          >
            <el-date-picker
              v-model="endDate"
              type="date"
              value-format="YYYY-MM-DD"
              :placeholder="$t('services.huangli.endDate')"
              class="w-full"
            />
          </el-form-item>
        </div>
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
        <div
          v-if="rangeInfo"
          class="range-info"
        >
          {{ $t('services.huangli.range', { start: rangeInfo.start, end: rangeInfo.end, days: rangeInfo.days }) }}
        </div>
        <div
          v-if="recommended.length"
          class="sec-title"
        >
          {{ $t('services.huangli.recommended') }}
        </div>
        <div
          v-if="recommended.length"
          class="days"
        >
          <div
            v-for="d in recommended"
            :key="d.date"
            class="day good"
          >
            <div class="d-date">
              {{ d.date }}
            </div>
            <div class="d-sub">
              {{ d.reason ? `${d.jian_chu} · ${d.reason}` : d.jian_chu }}
            </div>
          </div>
        </div>
        <p
          v-if="!recommended.length"
          class="hint"
        >
          {{ $t('services.huangli.noRecommended') }}
        </p>
        <div
          v-if="possible.length"
          class="sec-title"
        >
          {{ $t('services.huangli.possible') }}
        </div>
        <div
          v-if="possible.length"
          class="days"
        >
          <div
            v-for="d in possible"
            :key="d.date"
            class="day possible"
          >
            <div class="d-date">
              {{ d.date }}
            </div>
            <div class="d-sub">
              {{ d.reason ? `${d.jian_chu} · ${d.reason}` : d.jian_chu }}
            </div>
          </div>
        </div>
        <div
          v-if="unsuitable.length"
          class="sec-title"
        >
          {{ $t('services.huangli.unsuitable') }}
        </div>
        <div
          v-if="unsuitable.length"
          class="days"
        >
          <div
            v-for="d in unsuitable"
            :key="d.date"
            class="day bad"
          >
            <div class="d-date">
              {{ d.date }}
            </div>
            <div class="d-sub">
              {{ d.reason ? `${d.jian_chu} · ${d.reason}` : d.jian_chu }}
            </div>
          </div>
        </div>
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
import { engineHuangliDays, type HuangliDay } from '@/api/engine'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'

const EVENT_KEYS = [
  'wedding', 'engage', 'opening', 'sign', 'move', 'travel', 'build', 'exam',
  'medical', 'sacrifice', 'cleaning', 'renovation', 'bed_install', 'income', 'funeral',
] as const

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
const event = toRef(form, 'event')
const startDate = toRef(form, 'startDate')
const endDate = toRef(form, 'endDate')
const result = ref<HuangliDay[] | null>(null)
const loading = ref(false)
const error = ref('')

const formRef = ref<FormInstance>()
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

const days = computed(() => result.value ?? [])
const recommended = computed(() => days.value.filter(d => d.suitability === 'recommended'))
const possible = computed(() => days.value.filter(d => d.suitability === 'possible'))
const unsuitable = computed(() => days.value.filter(d => d.suitability === 'unsuitable'))
const rangeInfo = computed(() => {
  const list = days.value
  if (!list.length) return null
  return { start: list[0].date, end: list[list.length - 1].date, days: list.length }
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
    result.value = await engineHuangliDays(event.value, startDate.value, spanDays.value)
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
  // 择日属占卜域，复用 divination 专家部署。
  void openServiceChat(profile.id, 'divination', message => { error.value = message })
}

</script>

<style scoped>
.range-info { font-size: var(--font-size-label); color: var(--liki-text-2); margin-bottom: var(--space-3); font-variant-numeric: tabular-nums; }
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: var(--space-3); }
.days { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: var(--space-2); }
.day { border: 1px solid var(--liki-line); border-radius: var(--radius-md); padding: 10px; text-align: center; }
.day.good { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.day.possible { border-color: var(--liki-line); background: var(--liki-surface); }
.day.bad { border-color: var(--color-error-border); background: var(--color-danger-surface); }
.d-date { font-weight: 700; font-variant-numeric: tabular-nums; }
.d-sub { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
</style>
