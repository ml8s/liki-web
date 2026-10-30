<script setup lang="ts">
// huangli 页免费择日岛：事项 + 日期范围 → 黄历候选日；选中年/日 + 出生信息 →
// 八字与紫微流日证据。所有命理结论仍由引擎契约返回，前端不做推断。
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  baziFlowDayFor,
  ziweiFlowDayFor,
} from '@shared/domain/date-selection'
import { huangliDaysFor, type HuangliDays } from '@shared/domain/huangli'
import type { BaziLiuRiOutput } from '@shared/api/engine/bazi'
import type { ZiweiLiuRiData } from '@shared/api/engine/ziwei'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import ToolError from '@shared/components/tools/ToolError.vue'
import HuangliDaysPanel from '@shared/components/engine/huangli/HuangliDaysPanel.vue'
import BaziLiuRiPanel from '@shared/components/engine/bazi/BaziLiuRiPanel.vue'
import ZiweiLiuRiPanel from '@shared/components/engine/ziwei/ZiweiLiuRiPanel.vue'
import HuangliRangeFields from '@shared/components/huangli/HuangliRangeFields.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import { birthFormRules } from '@shared/components/form/birth-rules'
import { consultationBirthFromFields, type ConsultationContext } from '@shared/domain/consultation'
import { saveConsultationHandoff } from '@shared/utils/handoff'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import type { BirthInput } from '@shared/domain/birth'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    submit: string
    computing: string
    error: string
    resultIntro: string
    selectedHint: string
    personalTitle: string
    personalIntro: string
    personalCompute: string
    personalPrivacy: string
    cta: string
    privacy: string
  }
}>()

const { t } = useI18n()
const rules = birthFormRules(props.labels)

const form = reactive({ event: 'wedding', startDate: '', endDate: '' })
const input = ref<{ event: string; startDate: string; count: number } | null>(null)
const status = ref<EngineResultStatus>('idle')
const result = ref<HuangliDays | null>(null)
const rangeError = ref(false)
const birth = reactive<BirthFieldsValue>(createBirthFieldsValue())
const birthInput = ref<BirthInput | null>(null)
const selectedDate = ref('')
const baziStatus = ref<EngineResultStatus>('idle')
const ziweiStatus = ref<EngineResultStatus>('idle')
const baziFlowDay = ref<BaziLiuRiOutput | null>(null)
const ziweiFlowDay = ref<ZiweiLiuRiData | null>(null)

const spanDays = computed(() => {
  if (!form.startDate || !form.endDate) return 0
  return Math.round((new Date(form.endDate).getTime() - new Date(form.startDate).getTime()) / 86400000) + 1
})

const loading = computed(() =>
  !!input.value && (status.value === 'idle' || status.value === 'loading'),
)
const personalLoading = computed(() =>
  !!birthInput.value && (
    baziStatus.value === 'idle' || baziStatus.value === 'loading'
    || ziweiStatus.value === 'idle' || ziweiStatus.value === 'loading'
  ),
)
const personalReady = computed(() => !!birthInput.value && !!selectedDate.value)

function onData(value: unknown) {
  result.value = (value as HuangliDays | null) ?? null
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function onBaziFlowData(value: unknown) {
  baziFlowDay.value = (value as { day: BaziLiuRiOutput } | null)?.day ?? null
}

function onBaziFlowStatus(value: EngineResultStatus) {
  baziStatus.value = value
}

function onZiweiFlowData(value: unknown) {
  ziweiFlowDay.value = (value as { day: ZiweiLiuRiData } | null)?.day ?? null
}

function onZiweiFlowStatus(value: EngineResultStatus) {
  ziweiStatus.value = value
}

function loadHuangli(value: { event: string; startDate: string; count: number }) {
  return huangliDaysFor(value.event, value.startDate, value.count)
}

function loadBaziFlowDay(value: { birth: BirthInput; selectedDate: string }) {
  return baziFlowDayFor(value)
}

function loadZiweiFlowDay(value: { birth: BirthInput; selectedDate: string }) {
  return ziweiFlowDayFor(value)
}

function selectDay(date: string) {
  if (selectedDate.value === date) return
  selectedDate.value = date
  baziFlowDay.value = null
  ziweiFlowDay.value = null
  baziStatus.value = 'idle'
  ziweiStatus.value = 'idle'
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
  selectedDate.value = ''
  birthInput.value = null
  baziFlowDay.value = null
  ziweiFlowDay.value = null
  baziStatus.value = 'idle'
  ziweiStatus.value = 'idle'
  input.value = {
    event: form.event,
    startDate: form.startDate,
    count,
  }
}

function submitPersonal() {
  if (!isBirthInputReady(birth) || !selectedDate.value || personalLoading.value) return
  baziFlowDay.value = null
  ziweiFlowDay.value = null
  baziStatus.value = 'idle'
  ziweiStatus.value = 'idle'
  birthInput.value = birthInputFromFields(birth)
}

function saveHandoff() {
  const context: ConsultationContext = {
    version: 1,
    scenario: 'huangli',
    event: form.event,
    start_date: form.startDate,
    end_date: form.endDate,
    selected_date: selectedDate.value || undefined,
  }
  if (isBirthInputReady(birth)) context.birth = consultationBirthFromFields(birth)
  saveConsultationHandoff(context)
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
        <p class="tool-pan-intro">
          {{ labels.resultIntro }}
        </p>
        <HuangliDaysPanel
          :result="days"
          @select="selectDay"
        />
      </div>

      <template v-if="selectedDate">
        <div class="tool-pan">
          <h3 class="tool-pan-title">{{ labels.personalTitle }}</h3>
          <p class="tool-pan-intro">
            {{ labels.personalIntro }}
            <strong>{{ selectedDate }}</strong>
          </p>
          <FormShell
            form-class="tool-form"
            :model="birth"
            :rules="rules"
            :submit-label="personalLoading ? labels.computing : labels.personalCompute"
            :loading="personalLoading"
            @submit="submitPersonal"
          >
            <BirthFields
              :value="birth"
              :labels="labels"
              name-prefix="huangli-person"
              @update:patch="Object.assign(birth, $event)"
            />
          </FormShell>
          <p
            v-if="!personalReady"
            class="tool-pan-intro"
          >
            {{ labels.selectedHint }}
          </p>
          <p
            v-else
            class="tool-pan-intro"
          >
            {{ labels.personalPrivacy }}
          </p>
        </div>

        <EngineQueryPanel
          v-if="birthInput && selectedDate"
          resource="huangli.person.bazi.liuri"
          :input="{ birth: birthInput, selectedDate }"
          :load="loadBaziFlowDay"
          :enabled="!!birthInput && !!selectedDate"
          :fallback="labels.error"
          @update:data="onBaziFlowData"
          @update:status="onBaziFlowStatus"
        />

        <EngineQueryPanel
          v-if="birthInput && selectedDate"
          resource="huangli.person.ziwei.liuri"
          :input="{ birth: birthInput, selectedDate }"
          :load="loadZiweiFlowDay"
          :enabled="!!birthInput && !!selectedDate"
          :fallback="labels.error"
          @update:data="onZiweiFlowData"
          @update:status="onZiweiFlowStatus"
        />

        <div
          v-if="baziFlowDay || ziweiFlowDay"
          class="tool-pan"
        >
          <h3 class="tool-pan-title">{{ selectedDate }}</h3>
          <BaziLiuRiPanel
            v-if="baziFlowDay"
            :result="baziFlowDay"
          />
          <ZiweiLiuRiPanel
            v-if="ziweiFlowDay"
            :result="ziweiFlowDay"
          />
        </div>
      </template>

      <ToolCta
        :href="appUrl"
        :privacy="labels.privacy"
        @click="saveHandoff"
      >
        {{ labels.cta }}
      </ToolCta>
    </template>
  </EngineQueryPanel>
</template>
