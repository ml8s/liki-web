<script setup lang="ts">
// huangli 页免费择日岛：事项 + 日期范围 → 共享引擎 → 宜/平/忌分组展示。
// 结果组件与查询生命周期来自共享层（与 App 的 HuangliView 同源）。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { huangliDaysFor, type HuangliDays } from '@shared/domain/huangli'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import ToolError from '@shared/components/tools/ToolError.vue'
import HuangliDaysPanel from '@shared/components/engine/huangli/HuangliDaysPanel.vue'
import HuangliRangeFields from '@shared/components/huangli/HuangliRangeFields.vue'
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

const form = reactive({ event: 'wedding', startDate: '', endDate: '' })
const input = ref<{ event: string; startDate: string; count: number } | null>(null)
const status = ref<EngineResultStatus>('idle')
const result = ref<HuangliDays | null>(null)
const rangeError = ref(false)

const spanDays = computed(() => {
  if (!form.startDate || !form.endDate) return 0
  return Math.round((new Date(form.endDate).getTime() - new Date(form.startDate).getTime()) / 86400000) + 1
})

const loading = computed(() =>
  !!input.value && (status.value === 'idle' || status.value === 'loading'),
)

function onData(value: unknown) {
  result.value = (value as HuangliDays | null) ?? null
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function loadHuangli(value: { event: string; startDate: string; count: number }) {
  return huangliDaysFor(value.event, value.startDate, value.count)
}

function submit() {
  if (!form.startDate || !form.endDate || loading.value) return
  const count = spanDays.value
  if (form.endDate < form.startDate || count <= 0 || count > 30) {
    rangeError.value = true
    return
  }
  rangeError.value = false
  result.value = null
  input.value = {
    event: form.event,
    startDate: form.startDate,
    count,
  }
}
</script>

<template>
  <FormShell
    form-class="tool-form"
    :model="form"
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <HuangliRangeFields :value="form" @update:patch="Object.assign(form, $event)" />
  </FormShell>

  <ToolError :message="rangeError ? labels.error : ''" />

  <EngineQueryPanel
    v-if="input"
    resource="huangli.days"
    :input="input"
    :load="loadHuangli"
    :enabled="!!input"
    :fallback="labels.error"
    @update:data="onData"
    @update:status="onStatus"
  >
    <template #default="{ result: days }">
      <div class="tool-pan">
        <h3 class="tool-pan-title">
          {{ t('services.huangli.resultTitle', { event: t(`services.huangli.events.${form.event}`) }) }}
        </h3>
        <HuangliDaysPanel :result="days" />
      </div>
      <ToolCta
        :href="appUrl"
        :privacy="labels.privacy"
        @click="saveConsultationHandoff({
          version: 1,
          scenario: 'huangli',
          event: form.event,
          start_date: form.startDate,
          end_date: form.endDate,
        })"
      >
        {{ labels.cta }}
      </ToolCta>
    </template>
  </EngineQueryPanel>
</template>
