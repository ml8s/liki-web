<template>
  <section
    class="flow-fengshui-form"
    :aria-busy="loading"
  >
    <header class="flow-fengshui-form__header">
      <h2>{{ $t('services.fengshui.pageTitle') }}</h2>
      <p>{{ $t('services.fengshui.subtitle') }}</p>
    </header>

    <dl class="flow-fengshui-form__person">
      <div>
        <dt>{{ $t('chat.selectBirthProfile') }}</dt>
        <dd>{{ personLabel }}</dd>
      </div>
      <div>
        <dt>{{ $t('profiles.form.birthYear') }}</dt>
        <dd>{{ birthYear ?? '—' }}</dd>
      </div>
      <div>
        <dt>{{ $t('profiles.form.gender') }}</dt>
        <dd>{{ genderLabel }}</dd>
      </div>
    </dl>

    <FormShell
      :model="formValue"
      :submit-label="$t('services.fengshui.pageTitle')"
      :loading="loading"
      :disabled="!canSubmit"
      form-class="flow-fengshui-form__form"
      @submit="submit"
    >
      <FengshuiLayoutFields
        :value="formValue"
        @update:patch="patchLayout"
      />
    </FormShell>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import FormShell from '@shared/components/form/FormShell.vue'
import FengshuiLayoutFields, {
  type FengshuiLayoutValue,
} from '@shared/components/fengshui/FengshuiLayoutFields.vue'

const props = withDefaults(defineProps<{
  personLabel: string
  birthYear: number | null
  gender: '' | 'female' | 'male'
  loading?: boolean
}>(), {
  loading: false,
})

const emit = defineEmits<{
  submit: [value: FengshuiLayoutValue]
}>()

const { t } = useI18n()
const formValue = reactive<FengshuiLayoutValue>({
  zuoShan: 0,
  periodDate: new Date().toISOString().slice(0, 10),
  doorGua: '',
  masterGua: '',
  stoveGua: '',
})

const canSubmit = computed(() =>
  props.birthYear !== null
    && (props.gender === 'female' || props.gender === 'male')
    && Boolean(
      formValue.periodDate
      && formValue.doorGua
      && formValue.masterGua
      && formValue.stoveGua,
    ))

const genderLabel = computed(() => {
  if (props.gender === 'female') return t('profiles.female')
  if (props.gender === 'male') return t('profiles.male')
  return '—'
})

function patchLayout(patch: Partial<FengshuiLayoutValue>) {
  Object.assign(formValue, patch)
}

function submit() {
  if (!canSubmit.value || props.loading) return
  emit('submit', { ...formValue })
}
</script>

<style scoped>
.flow-fengshui-form {
  min-width: 0;
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
}

.flow-fengshui-form__header {
  margin-bottom: var(--space-3);
}

.flow-fengshui-form__header h2 {
  margin: 0;
  color: var(--liki-ink);
  font-size: var(--font-size-title-xs);
  font-weight: 650;
  letter-spacing: -0.02em;
}

.flow-fengshui-form__header p {
  margin: var(--space-1) 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
}

.flow-fengshui-form__person {
  margin: 0 0 var(--space-3);
  display: grid;
  gap: var(--space-2);
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
}

.flow-fengshui-form__person > div {
  min-width: 0;
}

.flow-fengshui-form__person dt {
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
}

.flow-fengshui-form__person dd {
  margin: 2px 0 0;
  color: var(--liki-ink);
  font-weight: 600;
  overflow-wrap: anywhere;
}

.flow-fengshui-form__form {
  min-width: 0;
}
</style>
