<script setup lang="ts">
// hepan 页免费合盘岛（营销站）：双方出生信息 → 共享资源包装 → 合盘盘面。
// 盘面组件与查询生命周期均来自共享层，与 App 同源。
import { computed, reactive, ref, watch } from 'vue'
import type { FormRules } from 'element-plus'
import { hepanFor, type Hepan } from '@shared/domain/hepan'
import type { BirthInput } from '@shared/domain/birth'
import type { EngineResultStatus } from '@shared/composables/useEngineResult'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import ToolCta from '@shared/components/tools/ToolCta.vue'
import HepanBirthFields from '@shared/components/hepan/HepanBirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import CheckboxField from '@shared/components/form/fields/CheckboxField.vue'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import BaziBondPanel from '@shared/components/engine/bazi/BaziBondPanel.vue'
import { consultationBirthFromFields, type ConsultationContext } from '@shared/domain/consultation'
import { HANDOFF_KEYS, saveConsultationHandoff, saveHandoff } from '@shared/utils/handoff'

const props = defineProps<{
  locale: string
  appUrl: string
  labels: BirthFieldLabels & {
    personA: string
    personB: string
    you: string
    submit: string
    computing: string
    error: string
    result: string
    resultIntro: string
    cta: string
    privacy: string
    consent: string
    consentRequired: string
    }
}>()

type PersonForm = BirthFieldsValue

const a = reactive<PersonForm>(createBirthFieldsValue())
const b = reactive<PersonForm>({ ...createBirthFieldsValue(), gender: 'female' })
const thirdPartyConsent = ref(false)
const rules: FormRules = {
  thirdPartyConsent: [{
    validator: (_rule, value: boolean, callback: (error?: Error) => void) => {
      if (!value) callback(new Error(props.labels.consentRequired))
      else callback()
    },
    trigger: 'change',
  }],
}
function patchPeople(patch: Partial<{ personA: unknown; personB: unknown }>) {
  if (patch.personA) Object.assign(a, patch.personA)
  if (patch.personB) Object.assign(b, patch.personB)
}

const hepanInput = ref<{ a: BirthInput; b: BirthInput } | null>(null)
const status = ref<EngineResultStatus>('idle')
const hepan = ref<Hepan | null>(null)
const consultationContext = ref<ConsultationContext | null>(null)

const loading = computed(() =>
  !!hepanInput.value && (status.value === 'idle' || status.value === 'loading'),
)

// 同意是计算前置事实；撤回后已有结果不能再作为继续咨询的上下文。
watch(thirdPartyConsent, consent => {
  if (!consent) {
    hepanInput.value = null
    hepan.value = null
    consultationContext.value = null
  }
})

function onHepanData(value: unknown) {
  hepan.value = (value as Hepan | null) ?? null
  if (hepan.value?.bond && hepan.value.chartA && hepan.value.chartB && thirdPartyConsent.value) {
    consultationContext.value = {
      version: 1,
      intent: 'hepan',
      second_birth: consultationBirthFromFields(b),
      third_party_consent: true,
    }
  } else {
    consultationContext.value = null
  }
}

function onStatus(value: EngineResultStatus) {
  status.value = value
}

function loadHepan(input: { a: BirthInput; b: BirthInput }) {
  return hepanFor(input.a, input.b)
}

function submit() {
  if (!thirdPartyConsent.value || !isBirthInputReady(a) || !isBirthInputReady(b) || loading.value) return
  hepan.value = null
  consultationContext.value = null
  hepanInput.value = {
    a: birthInputFromFields(a),
    b: birthInputFromFields(b),
  }
}

function continueToApp() {
  if (consultationContext.value) saveConsultationHandoff(consultationContext.value)
  // 公共工具的 A 方完整出生事实可先交接给建档页；B 方事实保留在 consultation handoff。
  saveHandoff(HANDOFF_KEYS.birth, { ...a })
}
</script>

<template>
  <FormShell
    form-class="tool-form tool-form--stack"
    :model="{ personA: a, personB: b, thirdPartyConsent }"
    :rules="rules"
    :submit-label="loading ? labels.computing : labels.submit"
    :loading="loading"
    @submit="submit"
  >
    <HepanBirthFields
      :value="{ personA: a, personB: b }"
      :labels="labels"
      :group-labels="{ personA: labels.personA, personB: labels.personB }"
      @update:patch="patchPeople"
    />
    <CheckboxField
      v-model="thirdPartyConsent"
      name="hepan-third-party-consent"
      prop="thirdPartyConsent"
      :required="true"
    >
      {{ labels.consent }}
    </CheckboxField>
  </FormShell>

  <EngineQueryPanel
    v-if="hepanInput"
    resource="hepan.bond"
    :input="hepanInput"
    :load="loadHepan"
    :enabled="!!hepanInput"
    :fallback="labels.error"
    @update:data="onHepanData"
    @update:status="onStatus"
  >
    <template #default="{ result }">
      <template v-if="result.bond && result.chartA && result.chartB">
        <section class="tool-pan">
          <h3 class="tool-pan-title">
            {{ labels.result }}
          </h3>
          <p class="tool-pan-intro">
            {{ labels.resultIntro }}
          </p>
          <BaziBondPanel
            :result="result.bond"
            :chart-a="result.chartA"
            :chart-b="result.chartB"
            :name-a="labels.you"
          />
        </section>
        <ToolCta
          :href="appUrl"
          :privacy="labels.privacy"
          @click="continueToApp"
        >
          {{ labels.cta }}
        </ToolCta>
      </template>
    </template>
  </EngineQueryPanel>
</template>
