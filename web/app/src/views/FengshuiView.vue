<template>
  <ServicePageShell
    :title="$t('services.fengshui.pageTitle')"
    :subtitle="$t('services.fengshui.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.fengshui.selectBirthProfile') }}
        </h2>
      </template>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
      >
        <el-form-item prop="profileId">
          <el-select
            v-model="profileId"
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
        {{ $t('services.fengshui.noBirthProfilePrefix') }}<router-link to="/birth-profiles/new">
          {{ $t('profiles.create') }}
        </router-link>{{ $t('services.fengshui.noBirthProfileSuffix') }}
      </p>
    </el-card>

    <template v-if="active">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <div class="app-card__header">
            <h2 class="app-card__title">
              {{ $t('services.fengshui.lifeGuaDirections') }}
            </h2>
            <el-button
              size="small"
              :disabled="!bazhai"
              @click="saveRecord"
            >
              {{ $t('records.save') }}
            </el-button>
          </div>
        </template>
        <EngineQueryPanel
          resource="fengshui.bazhai"
          :input="bazhaiInput"
          :load="loadBazhai"
          :enabled="active"
          :fallback="$t('services.fengshui.failed')"
          @update:data="onBazhaiData"
        >
          <template #default="{ result }">
            <BazhaiChartPanel
              :result="result"
              :layout="layoutResult"
            />
          </template>
        </EngineQueryPanel>
      </el-card>

      <el-card
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.fengshui.xuankongTitle') }}
          </h2>
        </template>
        <FengshuiLayoutFields
          :value="fengshuiLayout"
          @update:patch="patchFengshuiLayout"
        />
        <EngineQueryPanel
          resource="fengshui.xuankong"
          :input="xuankongInput"
          :load="loadXuankong"
          :enabled="active"
          :fallback="$t('services.fengshui.failed')"
        >
          <template #default="{ result }">
            <XuankongChartPanel :result="result" />
          </template>
        </EngineQueryPanel>
      </el-card>

      <el-card
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.fengshui.layoutTitle') }}
          </h2>
        </template>
        <EngineQueryPanel
          resource="fengshui.layout"
          :input="layoutInput"
          :load="loadLayout"
          :enabled="!!layoutInput"
          :fallback="$t('services.fengshui.failed')"
          @update:data="onLayoutData"
        >
          <template #default="{ result }">
            <BazhaiLayoutPanel :result="result" />
          </template>
        </EngineQueryPanel>
      </el-card>

      <AppConsultationCard
        :title="$t('services.fengshui.consultationTitle')"
        :description="$t('services.fengshui.consultationBody')"
        :locked-description="$t('services.fengshui.lockedBody')"
        :purchased="entitlements.purchased"
        @action="startChat"
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
import { firstQueryValue } from '@/utils/query'
import { computed, onMounted, reactive, ref, toRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { bazhaiChartFor, bazhaiLayoutFor, xuankongChartFor, type BazhaiChart, type BazhaiLayout } from '@shared/domain/fengshui'
import { useToolRecords } from '@/composables/useToolRecords'
import { birthYearFromDate } from '@shared/domain/birth'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import BazhaiLayoutPanel from '@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'
import XuankongChartPanel from '@shared/components/engine/fengshui/XuankongChartPanel.vue'
import FengshuiLayoutFields from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import type { ConsultationContext } from '@shared/domain/consultation'
import { consumeConsultationHandoff } from '@shared/utils/handoff'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()
const {
  records: toolRecords,
  error: recordsError,
  load: loadRecords,
  save: saveToolRecord,
  confirmRename: renameToolRecord,
  confirmRemove: removeToolRecord,
} = useToolRecords('bazhai')
void loadRecords()

// 表单模型保持稳定对象，v-model 写入同一引用，change 当拍校验才能读到新值。
const form = reactive({ profileId: '' })
const profileId = toRef(form, 'profileId')
const route = useRoute()
const router = useRouter()

// 档案选择同步 URL（?profile= 可分享/刷新保留）。
const initialProfile = firstQueryValue(route.query.profile)
if (initialProfile) form.profileId = initialProfile
watch(profileId, value => {
  const query = { ...route.query }
  if (value) query.profile = value
  else delete query.profile
  void router.replace({ query })
})
const bazhai = ref<BazhaiChart | null>(null)
const layoutResult = ref<BazhaiLayout | null>(null)
const error = ref('')

// 玄空飞星（坐山默认子山午向，立极日期默认今天）与八宅布局（门/主/灶三卦）。
const fengshuiLayout = reactive({
  zuoShan: 0,
  periodDate: new Date().toISOString().slice(0, 10),
  doorGua: '',
  masterGua: '',
  stoveGua: '',
})
const publicConsultation = consumeConsultationHandoff<ConsultationContext>('fengshui')
if (publicConsultation?.scenario === 'fengshui' && publicConsultation.layout) {
  fengshuiLayout.zuoShan = publicConsultation.layout.zuo_shan
  fengshuiLayout.periodDate = publicConsultation.layout.period_date
  fengshuiLayout.doorGua = publicConsultation.layout.door_gua ?? ''
  fengshuiLayout.masterGua = publicConsultation.layout.master_gua ?? ''
  fengshuiLayout.stoveGua = publicConsultation.layout.stove_gua ?? ''
}
type FengshuiLayoutPatch = Partial<typeof fengshuiLayout>
function patchFengshuiLayout(patch: FengshuiLayoutPatch) {
  Object.assign(fengshuiLayout, patch)
}

const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)
const birthYear = computed(() => {
  const date = profile.value?.birth.date
  return date ? birthYearFromDate(date) : null
})
const active = computed(() => !!profile.value && birthYear.value !== null)
const bazhaiInput = computed(() => active.value && profile.value && birthYear.value !== null
  ? { birthYear: birthYear.value, gender: profile.value.birth.gender }
  : null)
const xuankongInput = computed(() => active.value
  ? { periodDate: fengshuiLayout.periodDate, zuoShan: fengshuiLayout.zuoShan }
  : null)
const layoutInput = computed(() => {
  const chart = bazhai.value
  if (!chart || !fengshuiLayout.doorGua || !fengshuiLayout.masterGua || !fengshuiLayout.stoveGua) {
    return null
  }
  return {
    mingGua: chart.ming_gua.gua.name,
    doorGua: fengshuiLayout.doorGua,
    masterGua: fengshuiLayout.masterGua,
    stoveGua: fengshuiLayout.stoveGua,
  }
})

function loadBazhai(input: { birthYear: number; gender: string }) {
  return bazhaiChartFor(input.birthYear, input.gender)
}

function loadXuankong(input: { periodDate: string; zuoShan: number }) {
  return xuankongChartFor(input.periodDate, input.zuoShan)
}

function loadLayout(input: { mingGua: string; doorGua: string; masterGua: string; stoveGua: string }) {
  return bazhaiLayoutFor(input.mingGua, input.doorGua, input.masterGua, input.stoveGua)
}

function onBazhaiData(value: unknown) {
  bazhai.value = (value as BazhaiChart | null) ?? null
  if (!value) layoutResult.value = null
}

function onLayoutData(value: unknown) {
  layoutResult.value = (value as BazhaiLayout | null) ?? null
}

const formRef = ref<FormInstance>()
// 档案数据问题（canonical 信息缺失/出生年份越界）属字段级错误，显示在档案选择旁。
const rules = computed<FormRules>(() => ({
  profileId: [{
    validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
      const p = profiles.list.find(item => item.id === value)
      if (!value || !p) return callback(new Error(t('chat.selectBirthProfileFirst')))
      if (!p.birth.date) return callback(new Error(t('profiles.chart.errors.missingTime')))
      if (birthYearFromDate(p.birth.date) === null) {
        return callback(new Error(t('services.fengshui.invalidBirthYear')))
      }
      callback()
    },
    trigger: 'change',
  }],
}))

async function onProfileChange() {
  // Never keep the previous profile's chart while the new resource loads.
  bazhai.value = null
  layoutResult.value = null
  error.value = ''
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  // 有效档案：资源容器监听档案与表单输入，自动取数。
}

// URL 带回档案时（分享/刷新）自动出盘。
onMounted(() => {
  if (profileId.value) void onProfileChange()
})

function saveRecord() {
  const p = profile.value
  if (!bazhai.value || !p) return
  void saveToolRecord(
    t('records.titleBazhai', { nickname: p.nickname }),
    {
      profile_id: p.id,
      layout: { ...fengshuiLayout },
    },
  )
}

function openRecord(record: ToolRecord) {
  profileId.value = String(record.input.profile_id ?? '')
  const layout = (record.input.layout ?? {}) as Partial<typeof fengshuiLayout>
  if (layout.zuoShan !== undefined) fengshuiLayout.zuoShan = layout.zuoShan
  if (layout.periodDate) fengshuiLayout.periodDate = layout.periodDate
  if (layout.doorGua !== undefined) fengshuiLayout.doorGua = layout.doorGua
  if (layout.masterGua !== undefined) fengshuiLayout.masterGua = layout.masterGua
  if (layout.stoveGua !== undefined) fengshuiLayout.stoveGua = layout.stoveGua
  void onProfileChange()
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = t('services.fengshui.intent')
  const context: ConsultationContext = {
    version: 1,
    scenario: 'fengshui',
    layout: {
      zuo_shan: fengshuiLayout.zuoShan,
      period_date: fengshuiLayout.periodDate,
      door_gua: fengshuiLayout.doorGua,
      master_gua: fengshuiLayout.masterGua,
      stove_gua: fengshuiLayout.stoveGua,
    },
  }
  void openServiceChat(p.id, 'fengshui', message => { error.value = message }, context)
}
</script>
