<script setup lang="ts">
// hepan 页免费合盘岛（营销站）：双方出生信息 → 共享排盘链路 → 合盘盘面。
// 盘面组件来自共享层（web/shared/components/charts），与 App HepanView 同源；
// 表单用共享 tool-form 的纵向变体（.tool-form--stack + fieldset），不自定义字段样式。
import { reactive, ref, watch } from 'vue'
import type { FormRules } from 'element-plus'
import { hepanFor, type Hepan } from '@shared/domain/hepan'
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
import HepanPanel from '@shared/components/charts/HepanPanel.vue'
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
    cta: string
    privacy: string
    consent: string
    consentRequired: string
    }
}>()

type PersonForm = BirthFieldsValue

const a = reactive<PersonForm>(createBirthFieldsValue())
const b = reactive<PersonForm>({ ...createBirthFieldsValue(), gender: 'female' })
const loading = ref(false)
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

const failed = ref(false)
const hepan = ref<Hepan | null>(null)
const consultationContext = ref<ConsultationContext | null>(null)

// 同意是计算前置事实；撤回后已有结果不能再作为继续咨询的上下文。
watch(thirdPartyConsent, consent => {
  if (!consent) {
    hepan.value = null
    consultationContext.value = null
  }
})

async function submit() {
  if (!thirdPartyConsent.value || !isBirthInputReady(a) || !isBirthInputReady(b) || loading.value) return
  loading.value = true
  failed.value = false
  hepan.value = null
  try {
    const result = await hepanFor(birthInputFromFields(a), birthInputFromFields(b))
    if (!thirdPartyConsent.value) return
    hepan.value = result
    consultationContext.value = {
      version: 1,
      scenario: 'hepan',
      second_birth: consultationBirthFromFields(b),
      third_party_consent: true,
    }
  } catch {
    failed.value = true
  } finally {
    loading.value = false
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

  <p v-if="failed" class="tool-error" role="alert">{{ labels.error }}</p>

  <template v-if="hepan">
    <section class="tool-pan">
      <h3 class="tool-pan-title">
        {{ labels.result }}
      </h3>
      <HepanPanel
        :chart-a="hepan.chartA"
        :chart-b="hepan.chartB"
        :bond="hepan.bond"
        :name-a="labels.you"
      />
    </section>
    <a
      class="btn-purple tool-cta"
      :href="appUrl"
      @click="continueToApp"
    >{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
