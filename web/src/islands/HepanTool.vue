<script setup lang="ts">
// hepan 页免费合盘岛（营销站）：双方出生信息 → 共享排盘链路 → 合盘盘面。
// 盘面组件来自共享层（web/shared/components/charts），与 App HepanView 同源；
// 表单用共享 tool-form 的纵向变体（.tool-form--stack + fieldset），不自定义字段样式。
import { reactive, ref } from 'vue'
import { hepanFor, type Hepan } from '@shared/domain/hepan'
import HepanBirthFields from '@shared/components/hepan/HepanBirthFields.vue'
import FormShell from '@shared/components/form/FormShell.vue'
import {
  birthInputFromFields,
  createBirthFieldsValue,
  isBirthInputReady,
  type BirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import HepanPanel from '@shared/components/charts/HepanPanel.vue'

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
    }
}>()

type PersonForm = BirthFieldsValue

const a = reactive<PersonForm>(createBirthFieldsValue())
const b = reactive<PersonForm>({ ...createBirthFieldsValue(), gender: 'female' })
const loading = ref(false)
function patchPeople(patch: Partial<{ personA: unknown; personB: unknown }>) {
  if (patch.personA) Object.assign(a, patch.personA)
  if (patch.personB) Object.assign(b, patch.personB)
}

const failed = ref(false)
const hepan = ref<Hepan | null>(null)

async function submit() {
  if (!isBirthInputReady(a) || !isBirthInputReady(b) || loading.value) return
  loading.value = true
  failed.value = false
  hepan.value = null
  try {
    hepan.value = await hepanFor(birthInputFromFields(a), birthInputFromFields(b))
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <FormShell
    form-class="tool-form tool-form--stack"
    :model="{ personA: a, personB: b }"
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
    <a class="btn-purple tool-cta" :href="appUrl">{{ labels.cta }}</a>
    <p class="tool-privacy">{{ labels.privacy }}</p>
  </template>
</template>
