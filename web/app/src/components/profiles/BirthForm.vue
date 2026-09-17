<template>
  <el-form label-position="top" class="birth-form" @submit.prevent>
    <div class="step-label">基本信息</div>
    <el-form-item label="称呼">
      <el-input v-model="nickname" placeholder="如：妈妈 / 我本人" maxlength="40" />
    </el-form-item>
    <el-form-item label="性别">
      <el-radio-group v-model="gender">
        <el-radio-button value="female">女</el-radio-button>
        <el-radio-button value="male">男</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="出生日期">
      <div class="date-row">
        <el-radio-group v-model="calendar" size="small">
          <el-radio-button value="solar">公历</el-radio-button>
          <el-radio-button value="lunar">农历</el-radio-button>
        </el-radio-group>
        <el-date-picker
          v-if="calendar === 'solar'"
          v-model="solarDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="选择日期"
          :disabled-date="disableFuture"
          style="width: 100%"
        />
        <el-input v-else v-model="lunarDate" placeholder="如 1958-闰5-12 或 1990-5-12" />
      </div>
    </el-form-item>

    <div class="step-label">出生时间</div>
    <el-form-item label="时间是否确定">
      <el-radio-group v-model="birthTimeKnown">
        <el-radio-button value="known">精确</el-radio-button>
        <el-radio-button value="approximate">只知道时辰</el-radio-button>
        <el-radio-button value="unknown">不确定</el-radio-button>
      </el-radio-group>
    </el-form-item>
    <el-form-item v-if="birthTimeKnown === 'known'" label="精确时间（将做真太阳时修正）">
      <el-time-picker v-model="timeValue" format="HH:mm" value-format="HH:mm" placeholder="如 06:00" style="width: 100%" />
    </el-form-item>
    <el-form-item v-else-if="birthTimeKnown === 'approximate'" label="时辰（按已定时辰排，不校正）">
      <div class="shichen">
        <el-button v-for="item in shichen" :key="item.label" size="small" :type="birthTime === item.value ? 'primary' : 'default'" @click="birthTime = item.value">
          {{ item.label }}
        </el-button>
      </div>
    </el-form-item>
    <el-form-item v-else>
      <p class="hint">不确定也能算，之后可在对话中定盘。</p>
    </el-form-item>

    <div class="step-label">更多信息（可选）</div>
    <el-form-item label="出生地（城市）">
      <el-input v-model="birthPlace" placeholder="用于真太阳时修正，如：上海" />
    </el-form-item>
    <el-form-item label="关系">
      <el-select v-model="relation" placeholder="本人 / 家人 / 朋友…" clearable style="width: 100%">
        <el-option label="本人" value="本人" />
        <el-option label="家人" value="家人" />
        <el-option label="朋友" value="朋友" />
        <el-option label="其他" value="其他" />
      </el-select>
    </el-form-item>
    <el-form-item label="备注">
      <el-input v-model="note" type="textarea" :rows="2" placeholder="可选" />
    </el-form-item>
    <el-form-item>
      <el-checkbox v-model="thirdPartyConsent">我确认已获得该人的同意为其建档</el-checkbox>
    </el-form-item>

    <el-form-item>
      <el-button type="primary" size="large" :loading="submitting" class="submit" @click="submit">保存档案</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ProfileInput } from '@/api/profiles'

const props = withDefaults(defineProps<{ submitting?: boolean; initial?: Partial<ProfileInput> }>(), {})
const emit = defineEmits<{ submit: [ProfileInput] }>()

const nickname = ref(props.initial?.nickname ?? '')
const gender = ref<'male' | 'female'>(props.initial?.gender ?? 'female')
const calendar = ref<'solar' | 'lunar'>(props.initial?.calendar ?? 'solar')
const solarDate = ref(props.initial?.birth_date ?? '')
const lunarDate = ref(props.initial?.birth_date ?? '')
const birthTimeKnown = ref<ProfileInput['birth_time_known']>(props.initial?.birth_time_known ?? 'known')
const birthTime = ref(props.initial?.birth_time ?? '')
const timeValue = ref(props.initial?.birth_time ?? '')
const birthPlace = ref(props.initial?.birth_place ?? '')
const relation = ref(props.initial?.relation ?? '')
const note = ref(props.initial?.note ?? '')
const thirdPartyConsent = ref(props.initial?.third_party_consent ?? false)

const shichen = [
  { label: '子', value: '23:00' }, { label: '丑', value: '01:00' }, { label: '寅', value: '03:00' },
  { label: '卯', value: '05:00' }, { label: '辰', value: '07:00' }, { label: '巳', value: '09:00' },
  { label: '午', value: '11:00' }, { label: '未', value: '13:00' }, { label: '申', value: '15:00' },
  { label: '酉', value: '17:00' }, { label: '戌', value: '19:00' }, { label: '亥', value: '21:00' },
]

watch(timeValue, value => { if (value) birthTime.value = value })

function disableFuture(date: Date) {
  return date.getTime() > Date.now()
}

const birthDate = computed(() => (calendar.value === 'solar' ? solarDate.value : lunarDate.value))

function submit() {
  emit('submit', {
    nickname: nickname.value.trim(),
    gender: gender.value,
    calendar: calendar.value,
    birth_date: birthDate.value,
    birth_time: birthTime.value,
    birth_time_known: birthTimeKnown.value,
    birth_place: birthPlace.value.trim(),
    relation: relation.value,
    note: note.value.trim(),
    third_party_consent: thirdPartyConsent.value,
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
  background: var(--liki-gold);
}
.birth-form .el-form-item { margin-bottom: 16px; }
.date-row { display: flex; gap: 10px; width: 100%; align-items: center; }
.date-row :deep(.el-date-editor) { flex: 1; }
.shichen { display: grid; grid-template-columns: repeat(6, 1fr); gap: 6px; }
.shichen .el-button { margin: 0; }
.hint { color: var(--liki-text-2); font-size: 13px; margin: 0; }
.submit { width: 100%; }
@media (max-width: 480px) { .shichen { grid-template-columns: repeat(4, 1fr); } }
</style>