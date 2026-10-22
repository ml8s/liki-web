<template>
  <ServicePageShell
    :title="$t('services.hepan.title')"
    :subtitle="$t('services.hepan.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.hepan.step1') }}
        </h2>
      </template>
      <el-form
        ref="formARef"
        :model="formA"
        :rules="rulesA"
        label-position="top"
      >
        <el-form-item prop="profileAId">
          <el-select
            v-model="profileAId"
            :aria-label="$t('chat.selectBirthProfile')"
            :placeholder="$t('chat.selectBirthProfile')"
            class="w-full"
            @change="onProfileChange"
          >
            <el-option
              v-for="p in profiles.list"
              :key="p.id"
              :value="p.id"
              :label="p.nickname"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <p
        v-if="!profiles.list.length"
        class="hint"
      >
        {{ $t('services.hepan.noBirthProfilePrefix') }}<router-link to="/birth-profiles/new">
          {{ $t('profiles.create') }}
        </router-link>{{ $t('services.hepan.noBirthProfileSuffix') }}
      </p>
    </el-card>

    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.hepan.step2') }}
        </h2>
      </template>
      <el-form
        ref="formBRef"
        :model="b"
        :rules="rulesB"
        label-position="top"
        class="b-form"
        @submit.prevent="compute"
      >
        <BirthFields
          :value="b"
          :labels="birthLabels"
          @update:patch="patchB"
        />
        <el-button
          type="primary"
          native-type="submit"
          :loading="loading"
        >
          {{ $t('services.hepan.compute') }}
        </el-button>
      </el-form>
    </el-card>

    <template v-if="result">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <div class="app-card__header">
            <h2 class="app-card__title">
              {{ $t('services.hepan.result') }}
            </h2>
            <el-button
              size="small"
              @click="saveRecord"
            >
              {{ $t('records.save') }}
            </el-button>
          </div>
        </template>

        <HepanPanel
          :chart-a="chartA"
          :chart-b="bChart"
          :bond="result"
          :name-a="aName"
        />
      </el-card>

      <AppConsultationCard
        :title="$t('services.hepan.consultationTitle')"
        :description="$t('services.hepan.consultationBody')"
        :locked-description="$t('services.hepan.lockedBody')"
        :purchased="purchased"
        :action-label="$t('services.hepan.cta')"
        :locked-action-label="$t('commerce.activateNow')"
        to="/chat"
        @action="startInterpretation"
      />
    </template>

    <el-card
      v-if="toolRecords.length || recordsError"
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('records.title') }}
        </h2>
      </template>
      <RecordList
        :records="toolRecords"
        :error="recordsError"
        @open="openRecord"
        @rename="renameToolRecord"
        @remove="removeToolRecord"
      />
    </el-card>
  </ServicePageShell>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, reactive, ref, toRef } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { baziChartForChartInput, type BaziChartData } from '@shared/domain/natal'
import { hepanForChartInputs, type Hepan } from '@shared/domain/hepan'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import HepanPanel from '@shared/components/charts/HepanPanel.vue'
import BirthFields from '@shared/components/form/BirthFields.vue'
import {
  birthInputFromFields,
  createBirthFieldLabels,
  createBirthFieldsValue,
  type BirthFieldsValue,
} from '@shared/components/form/birth-fields'
import { chartInputFromBirth } from '@shared/domain/birth'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()
const {
  records: toolRecords,
  error: recordsError,
  load: loadRecords,
  save: saveToolRecord,
  confirmRename: renameToolRecord,
  confirmRemove: removeToolRecord,
} = useToolRecords('bond')
void loadRecords()


// 表单模型保持稳定对象，v-model 写入同一引用，change 当拍校验才能读到新值。
const formA = reactive({ profileAId: '' })
const profileAId = toRef(formA, 'profileAId')
const chartA = ref<BaziChartData | null>(null)
const bChart = ref<BaziChartData | null>(null)
const b = ref<BirthFieldsValue>({ ...createBirthFieldsValue(), gender: 'female' })
const result = ref<Hepan['bond'] | null>(null)
const loading = ref(false)
const error = ref('')

const formARef = ref<FormInstance>()
const formBRef = ref<FormInstance>()

const birthLabels = computed(() => createBirthFieldLabels(t))

const rulesA = computed<FormRules>(() => ({
  profileAId: [{ required: true, message: t('chat.selectBirthProfileFirst'), trigger: 'change' }],
}))
const rulesB = computed<FormRules>(() => ({
  date: [{ required: true, message: t('profiles.form.errors.birthDateRequired'), trigger: 'change' }],
  time: [{ required: true, message: t('profiles.form.errors.exactTimeRequired'), trigger: 'change' }],
  location: [{
    validator: (_rule: unknown, value: unknown, callback: (error?: Error) => void) => {
      if (value === null || value === undefined) callback(new Error(t('profiles.form.errors.birthPlaceRequired')))
      else callback()
    },
    trigger: 'change',
  }],
}))

function patchB(patch: Partial<BirthFieldsValue>) {
  b.value = { ...b.value, ...patch }
}

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const profileA = computed(() => profiles.list.find(p => p.id === profileAId.value) ?? null)
const aName = computed(() => profileA.value?.nickname ?? t('services.hepan.profileA'))

async function loadA() {
  error.value = ''
  result.value = null
  try {
    const p = profileA.value
    if (!p) return
    if (!p.birth.instant || !p.birth.location.timezone || p.birth.location.longitude == null) {
      error.value = t('profiles.chart.errors.missingTime')
      return
    }
    chartA.value = await baziChartForChartInput({
      instant: p.birth.instant,
      timezone: p.birth.location.timezone,
      longitude: p.birth.location.longitude,
      gender: p.birth.gender,
    })
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.detail.loadFailed'))
  }
}

function onProfileChange() {
  if (profileAId.value) void loadA()
}

async function compute() {
  error.value = ''
  const validA = await formARef.value?.validate().then(() => true).catch(() => false)
  const validB = await formBRef.value?.validate().then(() => true).catch(() => false)
  if (!validA || !validB) return
  result.value = null
  bChart.value = null
  loading.value = true
  try {
    const p = profileA.value
    if (!p) return
    if (!p.birth.instant || !p.birth.location.timezone || p.birth.location.longitude == null) {
      error.value = t('profiles.chart.errors.missingTime')
      return
    }
    const birthB = birthInputFromFields(b.value)
    const inputB = await chartInputFromBirth(birthB)
    const hepan = await hepanForChartInputs(
      {
        instant: p.birth.instant,
        timezone: p.birth.location.timezone,
        longitude: p.birth.location.longitude,
        gender: p.birth.gender,
      },
      { ...inputB, gender: birthB.gender },
    )
    chartA.value = hepan.chartA
    bChart.value = hepan.chartB
    result.value = hepan.bond
  } catch (err) {
    error.value = toUserMessage(err, t('services.hepan.failed'))
  } finally {
    loading.value = false
  }
}

function saveRecord() {
  if (!result.value) return
  void saveToolRecord(
    t('records.titleBond', { a: aName.value, b: b.value.date }),
    { profile_id: profileAId.value, b: { ...b.value } },
  )
}

function openRecord(record: ToolRecord) {
  profileAId.value = String(record.input.profile_id ?? '')
  const inputB = (record.input.b ?? {}) as Partial<BirthFieldsValue>
  // 保持 b 对象引用稳定：form 的 model 指向同一对象，替换引用会让校验读到旧值。
  Object.assign(b.value, {
    gender: inputB.gender ?? 'female',
    date: String(inputB.date ?? ''),
    time: String(inputB.time ?? ''),
    location: inputB.location ?? null,
  })
  chartA.value = null
  void compute()
}

async function startInterpretation() {
  const p = profileA.value
  if (!p) return
  const bBirthProfile = b.value
  const bInfo = t('services.hepan.otherBirthProfileSummary', {
    gender: bBirthProfile.gender === 'male' ? t('profiles.male') : t('profiles.female'),
    date: bBirthProfile.date,
    time: bBirthProfile.time,
    place: bBirthProfile.location?.place
      ? t('services.hepan.otherPlace', { place: bBirthProfile.location.place })
      : '',
  })
  chat.pendingIntent = t('services.hepan.intent', { a: p.nickname, info: bInfo })
  await openServiceChat(p.id, 'hepan', message => { error.value = message })
}
</script>

<style scoped>
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: var(--space-3); }
</style>
