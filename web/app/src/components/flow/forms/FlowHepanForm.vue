<template>
  <section
    class="flow-hepan-form"
    :aria-busy="loading"
  >
    <header class="flow-hepan-form__header">
      <h2>{{ $t('services.hepan.title') }}</h2>
      <p>{{ $t('services.hepan.subtitle') }}</p>
    </header>

    <div class="flow-hepan-form__people">
      <fieldset class="flow-hepan-form__segment">
        <legend>{{ $t('services.hepan.selfSegment') }}</legend>
        <p class="flow-hepan-form__person">
          <strong>{{ personLabel }}</strong>
        </p>
      </fieldset>

      <el-form
        ref="formRef"
        :model="formValue"
        :rules="rules"
        label-position="top"
        class="flow-hepan-form__form"
        @submit.prevent="submit"
      >
        <fieldset class="flow-hepan-form__segment">
          <legend>{{ $t('services.hepan.otherSegment') }}</legend>
          <BirthFields
            :value="formValue"
            :labels="labels"
            name-prefix="flow-hepan-other"
            @update:patch="patchSecondBirth"
          />
          <CheckboxField
            v-model="formValue.third_party_consent"
            name="flow-hepan-consent"
            prop="third_party_consent"
            :label="$t('profiles.form.consent')"
          />
        </fieldset>

        <el-button
          type="primary"
          native-type="submit"
          :loading="loading"
        >
          {{ $t('services.hepan.compute') }}
        </el-button>
      </el-form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FormInstance, FormRules } from 'element-plus'
import CheckboxField from '@shared/components/form/fields/CheckboxField.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import {
  createBirthFieldLabels,
  createBirthFieldsValue,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'

export interface FlowHepanFormValue {
  secondBirth: BirthFieldsValue
  thirdPartyConsent: true
}

const props = withDefaults(defineProps<{
  personLabel: string
  loading?: boolean
}>(), {
  loading: false,
})

const emit = defineEmits<{
  submit: [value: FlowHepanFormValue]
}>()

const { t } = useI18n()
const formRef = ref<FormInstance>()
const formValue = reactive<BirthFieldsValue & { third_party_consent: boolean }>({
  ...createBirthFieldsValue(),
  third_party_consent: false,
})
const labels = computed(() => createBirthFieldLabels(t))

const rules = computed<FormRules>(() => ({
  gender: [{ required: true, message: t('profiles.form.errors.genderRequired'), trigger: 'change' }],
  date: [{ required: true, message: t('profiles.form.errors.birthDateRequired'), trigger: 'change' }],
  time: [{ required: true, message: t('profiles.form.errors.exactTimeRequired'), trigger: 'change' }],
  location: [{
    validator: (_rule: unknown, value: unknown, callback: (error?: Error) => void) => {
      if (value === null || value === undefined) callback(new Error(t('profiles.form.errors.birthPlaceRequired')))
      else callback()
    },
    trigger: 'change',
  }],
  third_party_consent: [{
    validator: (_rule: unknown, value: boolean, callback: (error?: Error) => void) => {
      if (!value) callback(new Error(t('profiles.form.errors.thirdPartyConsentRequired')))
      else callback()
    },
    trigger: 'change',
  }],
}))

function patchSecondBirth(patch: Partial<BirthFieldsValue>) {
  Object.assign(formValue, patch)
}

async function submit() {
  if (props.loading) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  const secondBirth: BirthFieldsValue = { ...formValue }
  delete (secondBirth as { third_party_consent?: boolean }).third_party_consent
  emit('submit', {
    secondBirth,
    thirdPartyConsent: true,
  })
}
</script>

<style scoped>
.flow-hepan-form {
  min-width: 0;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
}

.flow-hepan-form__header {
  margin-bottom: var(--space-3);
}

.flow-hepan-form__header h2 {
  margin: 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-xs);
  font-weight: 650;
  letter-spacing: -0.02em;
}

.flow-hepan-form__header p,
.flow-hepan-form__person {
  margin: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}

.flow-hepan-form__header p { margin-top: var(--space-1); }
.flow-hepan-form__people {
  min-width: 0;
  display: grid;
  gap: var(--space-4);
}
.flow-hepan-form__segment {
  min-width: 0;
  margin: 0;
  padding: var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
}
.flow-hepan-form__segment legend {
  padding-inline: var(--space-1);
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
  font-weight: 600;
}
.flow-hepan-form__person {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--liki-ink);
}
.flow-hepan-form__form { min-width: 0; }
</style>
