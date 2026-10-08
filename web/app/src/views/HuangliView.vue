<template>
  <section class="service-view">
    <AppPageHeader
      :title="$t('services.huangli.pageTitle')"
      :subtitle="$t('services.huangli.subtitle')"
    />

    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.huangli.pageTitle') }}
        </h2>
      </template>
      <el-form
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
          <el-form-item :label="$t('services.huangli.startDate')">
            <el-date-picker
              v-model="startDate"
              type="date"
              value-format="YYYY-MM-DD"
              :placeholder="$t('services.huangli.startDate')"
              class="w-full"
            />
          </el-form-item>
          <el-form-item :label="$t('services.huangli.endDate')">
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
          :disabled="!event || !startDate || !endDate"
        >
          {{ $t('services.huangli.calculate') }}
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
            {{ $t('services.huangli.resultTitle', { event: eventLabel(result.event || event) }) }}
          </h2>
        </template>
        <div
          v-if="result.range"
          class="range-info"
        >
          {{ $t('services.huangli.range', { start: result.range.start_date, end: result.range.end_date, days: result.range.days }) }}
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
              {{ d.jian_chu ? `${d.jian_chu} · ` : '' }}{{ d.reason || '' }}
            </div>
            <div
              v-if="d.warnings?.length"
              class="d-warn"
            >
              {{ d.warnings.join($t('common.listSeparator')) }}
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
              {{ d.jian_chu ? `${d.jian_chu} · ` : '' }}{{ d.reason || '' }}
            </div>
            <div
              v-if="d.warnings?.length"
              class="d-warn"
            >
              {{ d.warnings.join($t('common.listSeparator')) }}
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
              {{ d.reason || $t('services.huangli.unsuitable') }}
            </div>
            <div
              v-if="d.warnings?.length"
              class="d-warn"
            >
              {{ d.warnings.join($t('common.listSeparator')) }}
            </div>
          </div>
        </div>
        <div
          v-if="result.safety_advisory"
          class="safety-box"
        >
          <el-alert
            :title="result.safety_advisory"
            type="warning"
            :closable="false"
            show-icon
          />
        </div>
      </el-card>

      <AppConsultationCard
        :title="$t('services.huangli.consultationTitle')"
        :description="$t('services.huangli.consultationBody')"
        :locked-description="$t('services.huangli.lockedBody')"
        :purchased="entitlements.purchased && profiles.list.length > 0"
        :action-label="$t('services.huangli.consultationTitle')"
        :locked-action-label="profiles.list.length ? $t('commerce.activateNow') : $t('profiles.create')"
        :to="profiles.list.length ? '/chat' : '/profiles/new'"
        @action="startChat"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { engineHuangliDays, type HuangliDaysResult } from '@/api/engine'
import { huangliApiEventLabel } from '@/utils/huangliEvents'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const EVENT_KEYS = [
  'wedding', 'engage', 'opening', 'sign', 'move', 'travel', 'build', 'exam',
  'medical', 'sacrifice', 'cleaning', 'renovation', 'bed_install', 'income', 'funeral',
] as const

const router = useRouter()
const { t } = useI18n()
const { profiles, entitlements, chat } = useServiceView()

const event = ref('wedding')
const startDate = ref('')
const endDate = ref('')
const result = ref<HuangliDaysResult | null>(null)
const loading = ref(false)
const error = ref('')

const recommended = computed(() => result.value?.recommended ?? [])
const possible = computed(() => result.value?.candidates?.filter(d => d.suitability === 'possible') ?? [])
const unsuitable = computed(() => result.value?.unsuitable ?? [])

function eventLabel(key: string): string {
  return t(`services.huangli.events.${key}`)
}


async function calc() {
  if (!event.value || !startDate.value || !endDate.value) return
  if (startDate.value > endDate.value) {
    error.value = t('services.huangli.endBeforeStart')
    return
  }
  const spanDays = Math.round((new Date(endDate.value).getTime() - new Date(startDate.value).getTime()) / 86400000) + 1
  if (spanDays > 120) {
    error.value = t('services.huangli.rangeTooLong', { days: spanDays })
    return
  }
  loading.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await engineHuangliDays(huangliApiEventLabel(event.value), event.value, startDate.value, endDate.value)
  } catch (err) {
    error.value = toUserMessage(err, t('services.huangli.failed'))
  } finally {
    loading.value = false
  }
}

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  chat.pendingIntent = t('services.huangli.intent', { event: eventLabel(event.value) })
  void chat.ensureSession(profile.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, t('chat.createFailed')) })
}

</script>

<style scoped>
.range-info { font-size: 13px; color: var(--liki-text-2); margin-bottom: 12px; }
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
.days { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 8px; }
.day { border: 1px solid var(--liki-line); border-radius: 8px; padding: 10px; text-align: center; }
.day.good { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.day.possible { border-color: var(--liki-line); background: var(--liki-surface); }
.day.bad { border-color: var(--el-color-danger-light-7); background: var(--el-color-danger-light-9); }
.d-date { font-weight: 700; }
.d-sub { font-size: 12px; color: var(--liki-text-2); margin-top: 4px; }
.d-warn { font-size: 11px; color: var(--el-color-warning); margin-top: 4px; }
.safety-box { margin-top: 14px; }
</style>
