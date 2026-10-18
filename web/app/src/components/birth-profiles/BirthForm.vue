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

    <el-form-item
      :label="$t('profiles.form.gender')"
      prop="gender"
    >
      <el-radio-group v-model="form.gender">
        <el-radio-button value="female">
          {{ $t('profiles.female') }}
        </el-radio-button>
        <el-radio-button value="male">
          {{ $t('profiles.male') }}
        </el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item
      :label="$t('profiles.form.birthDate')"
      prop="birth_date"
    >
      <div class="date-row">
        <el-radio-group
          v-model="form.calendar"
          size="small"
        >
          <el-radio-button value="solar">
            {{ $t('profiles.form.solar') }}
          </el-radio-button>
          <el-radio-button value="lunar">
            {{ $t('profiles.form.lunar') }}
          </el-radio-button>
        </el-radio-group>

        <el-date-picker
          v-if="form.calendar === 'solar'"
          v-model="form.birth_date"
          :aria-label="$t('profiles.form.birthDate')"
          type="date"
          value-format="YYYY-MM-DD"
          :placeholder="$t('profiles.form.selectDate')"
          :disabled-date="disabledDate"
          class="w-full"
        />
        <el-input
          v-else
          v-model="form.birth_date"
          name="lunar-birth-date"
          autocomplete="off"
          spellcheck="false"
          :placeholder="$t('profiles.form.lunarPlaceholder')"
        />
      </div>
    </el-form-item>

    <div class="step-label">
      {{ $t('profiles.form.birthTime') }}
    </div>

    <el-form-item
      :label="$t('profiles.form.timeKnown')"
      prop="birth_time_known"
    >
      <el-radio-group v-model="form.birth_time_known">
        <el-radio-button value="known">
          {{ $t('profiles.form.known') }}
        </el-radio-button>
        <el-radio-button value="approximate">
          {{ $t('profiles.form.approximate') }}
        </el-radio-button>
        <el-radio-button value="unknown">
          {{ $t('profiles.form.unknown') }}
        </el-radio-button>
      </el-radio-group>
    </el-form-item>

    <template v-if="form.birth_time_known === 'known'">
      <el-form-item
        :label="$t('profiles.form.exactTime')"
        prop="birth_time"
      >
        <el-time-picker
          v-model="form.birth_time"
          format="HH:mm"
          value-format="HH:mm"
          :placeholder="$t('profiles.form.timePlaceholder')"
          class="w-full"
        />
      </el-form-item>

      <el-form-item :label="$t('profiles.form.birthPlace')">
        <el-input
          v-model="form.birth_place"
          name="birth-place"
          autocomplete="off"
          spellcheck="false"
          :placeholder="$t('profiles.form.birthPlacePlaceholder')"
        />
      </el-form-item>
    </template>

    <el-form-item
      v-else-if="form.birth_time_known === 'approximate'"
      :label="$t('profiles.form.shichenLabel')"
    >
      <div class="shichen">
        <el-button
          v-for="item in shichen"
          :key="item.key"
          size="small"
          :type="form.birth_time === item.value ? 'primary' : 'default'"
          @click="form.birth_time = item.value"
        >
          <span class="shichen-name">{{ $t(`profiles.form.shichen.${item.key}`) }}</span>
          <span class="shichen-range">{{ item.range }}</span>
        </el-button>
      </div>
    </el-form-item>

    <el-form-item v-else>
      <p class="hint">
        {{ $t('profiles.form.unknownHint') }}
      </p>
    </el-form-item>

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
import type { BirthProfileInput } from '@/api/birth-profiles'
import { useFormDraftsStore } from '@/stores/form-drafts-store'

const props = withDefaults(defineProps<{
  submitting?: boolean
  initial?: Partial<BirthProfileInput>
  draftKey?: string
}>(), {})
const emit = defineEmits<{ submit: [BirthProfileInput]; dirty: [boolean] }>()
const { t } = useI18n()

type Gender = BirthProfileInput['gender'] | ''
type TimeKnown = BirthProfileInput['birth_time_known']
type BirthFormValues = {
  nickname: string
  gender: Gender
  calendar: BirthProfileInput['calendar']
  birth_date: string
  birth_time_known: TimeKnown
  birth_time: string
  birth_place: string
  note: string
  third_party_consent: boolean
}
type BirthFormDraft = Omit<BirthFormValues, 'third_party_consent'>

function formValuesFromInitial(initial?: Partial<BirthProfileInput>): BirthFormValues {
  return {
    nickname: initial?.nickname ?? '',
    gender: initial?.gender ?? '',
    calendar: initial?.calendar ?? 'solar',
    birth_date: initial?.birth_date ?? '',
    birth_time_known: initial?.birth_time_known ?? 'known',
    birth_time: initial?.birth_time ?? '',
    birth_place: initial?.birth_place ?? '',
    note: initial?.note ?? '',
    third_party_consent: initial?.third_party_consent ?? false,
  }
}

function mergeBirthFormDraft(initial?: Partial<BirthProfileInput>, draft?: Partial<BirthFormDraft> | null): BirthFormValues {
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
    calendar: values.calendar,
    birth_date: values.birth_date,
    birth_time_known: values.birth_time_known,
    birth_time: values.birth_time,
    birth_place: values.birth_place,
    note: values.note,
  }
}

const drafts = useFormDraftsStore()
const savedDraft = computed(() => props.draftKey ? drafts.get<BirthFormDraft>(props.draftKey) : null)
const form = reactive(mergeBirthFormDraft(props.initial, savedDraft.value))

if (savedDraft.value) emit('dirty', true)

const formRef = ref<FormInstance>()

function validateBirthDate(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value) return callback(new Error(t('profiles.form.errors.birthDateRequired')))
  if (form.calendar === 'lunar' && !/^\d{4}-闰?\d{1,2}-\d{1,2}$/.test(value)) {
    return callback(new Error(t('profiles.form.errors.lunarFormat')))
  }
  callback()
}

const rules = computed<FormRules>(() => ({
  nickname: [{ required: true, whitespace: true, message: t('profiles.form.errors.nicknameRequired'), trigger: 'blur' }],
  gender: [{ required: true, message: t('profiles.form.errors.genderRequired'), trigger: 'change' }],
  birth_date: [
    { required: true, message: t('profiles.form.errors.birthDateRequired'), trigger: 'change' },
    { validator: validateBirthDate, trigger: ['blur', 'change'] },
  ],
  birth_time_known: [{ required: true, message: t('profiles.form.errors.timeTypeRequired'), trigger: 'change' }],
  birth_time: [
    {
      required: form.birth_time_known === 'known',
      message: t('profiles.form.errors.exactTimeRequired'),
      trigger: 'change',
    },
    {
      pattern: /^\d{2}:\d{2}$/,
      message: t('profiles.form.errors.timeFormat'),
      trigger: 'change',
    },
  ],
}))

// value 为时辰起点（提交值），range 仅用于展示时间区间；数字区间不参与翻译。
const shichen = [
  { key: 'zi', value: '23:00', range: '23:00–01:00' }, { key: 'chou', value: '01:00', range: '01:00–03:00' }, { key: 'yin', value: '03:00', range: '03:00–05:00' },
  { key: 'mao', value: '05:00', range: '05:00–07:00' }, { key: 'chen', value: '07:00', range: '07:00–09:00' }, { key: 'si', value: '09:00', range: '09:00–11:00' },
  { key: 'wu', value: '11:00', range: '11:00–13:00' }, { key: 'wei', value: '13:00', range: '13:00–15:00' }, { key: 'shen', value: '15:00', range: '15:00–17:00' },
  { key: 'you', value: '17:00', range: '17:00–19:00' }, { key: 'xu', value: '19:00', range: '19:00–21:00' }, { key: 'hai', value: '21:00', range: '21:00–23:00' },
]

function disabledDate(date: Date) {
  return date.getTime() > Date.now() || date.getFullYear() < 1900
}

// 快照仅含表单字段，与 currentSnapshot() 同构——否则 dirty 恒为 true。
function snapshotOf(profile: typeof props.initial) {
  return JSON.stringify({
    nickname: profile?.nickname ?? '',
    gender: profile?.gender ?? '',
    calendar: profile?.calendar ?? '',
    birth_date: profile?.birth_date ?? '',
    birth_time_known: profile?.birth_time_known ?? '',
    birth_time: profile?.birth_time ?? '',
    birth_place: profile?.birth_place ?? '',
    note: profile?.note ?? '',
    third_party_consent: profile?.third_party_consent ?? false,
  })
}

const initialSnapshot = snapshotOf(props.initial)

function currentSnapshot() {
  return JSON.stringify({
    nickname: form.nickname,
    gender: form.gender,
    calendar: form.calendar,
    birth_date: form.birth_date,
    birth_time_known: form.birth_time_known,
    birth_time: form.birth_time,
    birth_place: form.birth_place,
    note: form.note,
    third_party_consent: form.third_party_consent,
  })
}

const isDirty = computed(() => currentSnapshot() !== initialSnapshot)

watch(form, () => {
  const dirty = isDirty.value
  emit('dirty', dirty)

  if (!props.draftKey) return
  if (!dirty) {
    drafts.clear(props.draftKey)
    return
  }

  drafts.set(props.draftKey, birthFormDraftFromValues(form))
}, { deep: true })

function clearDraft() {
  if (props.draftKey) drafts.clear(props.draftKey)
}

defineExpose({
  clearDraft,
})

async function submit() {
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) {
    void nextTick(() => {
      // .el-radio-group 容器不可聚焦：必须落到组内的原生 input（radio-button 用
      // .el-radio-button__original-radio），否则聚焦无效、键盘用户定位不到错误。
      const target = document.querySelector<HTMLElement>(
        '.el-form-item.is-error .el-radio__original, .el-form-item.is-error .el-radio-button__original-radio, .el-form-item.is-error input, .el-form-item.is-error textarea',
      )
      target?.focus()
    })
    return
  }

  emit('submit', {
    nickname: form.nickname.trim(),
    gender: form.gender || 'female',
    calendar: form.calendar,
    birth_date: form.birth_date,
    birth_time: form.birth_time,
    birth_time_known: form.birth_time_known,
    birth_place: form.birth_time_known === 'known' ? form.birth_place.trim() : '',
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
.date-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}
.date-row :deep(.el-date-editor) { flex: 1; }
.shichen { display: grid; grid-template-columns: repeat(6, 1fr); gap: 6px; }
.shichen .el-button { margin: 0; height: auto; padding: var(--space-1) 2px; line-height: 1.25; }
.shichen :deep(.el-button > span) { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.shichen-range { font-size: var(--font-size-micro); font-weight: 400; opacity: 0.8; font-variant-numeric: tabular-nums; }
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: 0; }
.submit { width: 100%; }
@media (max-width: 480px) { .shichen { grid-template-columns: repeat(4, 1fr); } }
</style>
