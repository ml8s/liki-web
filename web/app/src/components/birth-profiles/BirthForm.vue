<template>
  <el-form
    ref="formRef"
    :model="form"
    :rules="rules"
    label-position="top"
    class="birth-form"
    @submit.prevent="submit"
  >
    <div class="step-label">
      {{ $t('profiles.form.basic') }}
    </div>

    <el-form-item
      :label="$t('profiles.form.nickname')"
      prop="nickname"
    >
      <el-input
        v-model="form.nickname"
        name="nickname"
        autocomplete="off"
        spellcheck="false"
        :placeholder="$t('profiles.form.nicknamePlaceholder')"
        maxlength="40"
      />
    </el-form-item>

    <BirthFields
      :value="birthFields"
      :labels="birthLabels"
      :initial-place="initialBirthPlace"
      @update:patch="patchBirthFields"
    />

    <div class="step-label">
      {{ $t('profiles.form.more') }}
    </div>

    <el-form-item :label="$t('profiles.form.note')">
      <el-input
        v-model="form.note"
        type="textarea"
        :rows="2"
        name="note"
        autocomplete="off"
        :placeholder="$t('profiles.form.notePlaceholder')"
      />
    </el-form-item>

    <el-form-item prop="third_party_consent">
      <el-checkbox v-model="form.third_party_consent">
        {{ $t('profiles.form.consent') }}
      </el-checkbox>
    </el-form-item>

    <el-form-item>
      <el-button
        type="primary"
        size="large"
        :loading="submitting"
        native-type="submit"
        class="submit"
      >
        {{ $t('profiles.form.save') }}
      </el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BirthProfile, BirthProfileInput } from '@/api/birth-profiles'
import BirthFields from '@shared/components/form/BirthFields.vue'
import {
  createBirthFieldLabels,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import { useFormDraftsStore } from '@shared/stores/form-drafts'
const props = withDefaults(defineProps<{
  submitting?: boolean
  initial?: Partial<BirthProfile> | Partial<BirthProfileInput>
  draftKey?: string
}>(), {})
const emit = defineEmits<{ submit: [BirthProfileInput]; dirty: [boolean] }>()
const { t } = useI18n()

type Gender = BirthProfileInput['birth']['gender'] | ''

interface BirthFormValues {
  nickname: string
  gender: Gender
  date: string
  time: string
  location: BirthFieldsValue['location']
  note: string
  third_party_consent: boolean
}

type BirthFormDraft = Omit<BirthFormValues, 'third_party_consent'>

function formValuesFromInitial(initial?: Partial<BirthProfile> | Partial<BirthProfileInput>): BirthFormValues {
  return {
    nickname: initial?.nickname ?? '',
    gender: initial?.birth?.gender ?? '',
    date: initial?.birth?.date ?? '',
    time: initial?.birth?.time ?? '',
    location: initial?.birth?.location?.longitude != null && initial?.birth?.location?.timezone
      ? {
          place: initial?.birth?.location?.place ?? '',
          ...(initial.birth.location.place_id ? { place_id: initial.birth.location.place_id } : {}),
          longitude: initial.birth.location.longitude,
          timezone: initial.birth.location.timezone,
          source: 'saved',
        }
      : null,
    note: initial?.note ?? '',
    third_party_consent: initial?.third_party_consent ?? false,
  }
}

function mergeBirthFormDraft(
  initial?: Partial<BirthProfile> | Partial<BirthProfileInput>,
  draft?: Partial<BirthFormDraft> | null,
): BirthFormValues {
  const values = formValuesFromInitial(initial)
  if (!draft) return values
  return {
    ...values,
    ...Object.fromEntries(Object.entries(draft).filter(([, value]) => value !== undefined)),
  } as BirthFormValues
}

function birthFormDraftFromValues(values: BirthFormValues): BirthFormDraft {
  return {
    nickname: values.nickname,
    gender: values.gender,
    date: values.date,
    time: values.time,
    location: values.location,
    note: values.note,
  }
}

const drafts = useFormDraftsStore()
const savedDraft = computed(() => props.draftKey ? drafts.get<BirthFormDraft>(props.draftKey) : null)
const form = reactive(mergeBirthFormDraft(props.initial, savedDraft.value))

if (savedDraft.value) emit('dirty', true)

const formRef = ref<FormInstance>()
const birthLabels = computed(() => createBirthFieldLabels(t))
const initialBirthPlace = props.initial?.birth?.location?.place ?? ''

function validateBirthDate(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return callback(new Error(t('profiles.form.errors.birthDateRequired')))
  }
  callback()
}

function validateLocation(_rule: unknown, value: BirthFieldsValue['location'], callback: (error?: Error) => void) {
  if (!value) return callback(new Error(t('profiles.form.errors.birthPlaceRequired')))
  callback()
}

const rules = computed<FormRules>(() => ({
  nickname: [{ required: true, whitespace: true, message: t('profiles.form.errors.nicknameRequired'), trigger: 'blur' }],
  gender: [{ required: true, message: t('profiles.form.errors.genderRequired'), trigger: 'change' }],
  date: [
    { required: true, message: t('profiles.form.errors.birthDateRequired'), trigger: 'change' },
    { validator: validateBirthDate, trigger: ['blur', 'change'] },
  ],
  time: [
    {
      required: true,
      message: t('profiles.form.errors.exactTimeRequired'),
      trigger: 'change',
    },
    { pattern: /^\d{2}:\d{2}$/, message: t('profiles.form.errors.timeFormat'), trigger: 'change' },
  ],
  location: [{ validator: validateLocation, trigger: 'change' }],
  third_party_consent: [{
    validator: (_rule: unknown, value: boolean, callback: (error?: Error) => void) => {
      if (!value) callback(new Error(t('profiles.form.errors.thirdPartyConsentRequired')))
      else callback()
    },
    trigger: 'change',
  }],
}))

function patchBirthFields(patch: Partial<BirthFieldsValue>) {
  if (patch.gender !== undefined) form.gender = patch.gender
  if (patch.date !== undefined) form.date = patch.date
  if (patch.time !== undefined) form.time = patch.time
  if (patch.location !== undefined) form.location = patch.location
}

const birthFields = computed<BirthFieldsValue>(() => ({
  gender: form.gender,
  date: form.date,
  time: form.time,
  location: form.location,
}))

function snapshotOf(profile: typeof props.initial) {
  return JSON.stringify({
    nickname: profile?.nickname ?? '',
    gender: profile?.birth?.gender ?? '',
    date: profile?.birth?.date ?? '',
    time: profile?.birth?.time ?? '',
    location: profile?.birth?.location?.longitude != null && profile?.birth?.location?.timezone
      ? [profile.birth.location.place, profile.birth.location.longitude, profile.birth.location.timezone]
      : null,
    note: profile?.note ?? '',
    third_party_consent: profile?.third_party_consent ?? false,
  })
}

const initialSnapshot = snapshotOf(props.initial)

function currentSnapshot() {
  return JSON.stringify({
    nickname: form.nickname,
    gender: form.gender,
    date: form.date,
    time: form.time,
    location: form.location,
    note: form.note,
    third_party_consent: form.third_party_consent,
  })
}

const isDirty = computed(() => currentSnapshot() !== initialSnapshot)

watch(form, () => {
  emit('dirty', isDirty.value)
  if (!props.draftKey) return
  if (!isDirty.value) drafts.clear(props.draftKey)
  else drafts.set(props.draftKey, birthFormDraftFromValues(form))
}, { deep: true })

function clearDraft() {
  if (props.draftKey) drafts.clear(props.draftKey)
}

defineExpose({ clearDraft })

async function submit() {
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) {
    void nextTick(() => {
      const target = document.querySelector<HTMLElement>(
        '.el-form-item.is-error .el-radio__original, .el-form-item.is-error .el-radio-button__original-radio, .el-form-item.is-error input, .el-form-item.is-error textarea',
      )
      target?.focus()
    })
    return
  }

  emit('submit', {
    nickname: form.nickname.trim(),
    birth: {
      gender: form.gender || 'female',
      date: form.date,
      time: form.time,
      location: {
        place: form.location?.place ?? '',
        ...(form.location?.place_id ? { place_id: form.location.place_id } : {}),
        longitude: form.location?.longitude ?? 0,
        timezone: form.location?.timezone ?? '',
      },
    },
    note: form.note.trim(),
    third_party_consent: form.third_party_consent,
  })
}
</script>

<style scoped>
.step-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--liki-font-serif);
  font-weight: 700;
  margin: 8px 0 14px;
  color: var(--liki-ink);
}

.step-label::before {
  content: '';
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--liki-primary);
}

.birth-form .el-form-item { margin-bottom: 16px; }
.submit { width: 100%; }
</style>
