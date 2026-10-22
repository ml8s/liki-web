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

    <el-card
      v-if="loading"
      class="app-card app-card--highlight"
      shadow="never"
      aria-busy="true"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.fengshui.lifeGuaDirections') }}
        </h2>
      </template>
      <el-skeleton
        :rows="4"
        animated
      />
    </el-card>

    <template v-if="bazhai">
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
              @click="saveRecord"
            >
              {{ $t('records.save') }}
            </el-button>
          </div>
        </template>
        <MingGuaSection :bazhai="bazhai" />
        <LiuNianFlySection
          v-if="bazhai.liu_nian_xing"
          :liu-nian="bazhai.liu_nian_xing"
        />
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
        <XuankongPanel
          v-loading="xuankongLoading"
          :chart="xuankong"
        />
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
        <BazhaiLayoutPanel
          v-loading="layoutLoading"
          :layout="layoutResult"
          :bazhai="bazhai"
        />
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
import { toUserMessage } from '@/utils/errors'
import { firstQueryValue } from '@/utils/query'
import { computed, onMounted, reactive, ref, toRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { bazhaiChartFor, bazhaiLayoutFor, xuankongChartFor, type BazhaiChart, type BazhaiLayout, type XuankongChart } from '@shared/domain/fengshui'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import MingGuaSection from '@shared/components/paipan/MingGuaSection.vue'
import LiuNianFlySection from '@shared/components/paipan/LiuNianFlySection.vue'
import XuankongPanel from '@shared/components/paipan/XuankongPanel.vue'
import BazhaiLayoutPanel from '@shared/components/paipan/BazhaiLayoutPanel.vue'
import FengshuiLayoutFields from '@shared/components/fengshui/FengshuiLayoutFields.vue'

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
const loading = ref(false)
const error = ref('')

// 玄空飞星（坐山默认子山午向，立极日期默认今天）与八宅布局（门/主/灶三卦）。
const xuankong = ref<XuankongChart | null>(null)
const xuankongLoading = ref(false)
const fengshuiLayout = reactive({
  zuoShan: 0,
  periodDate: new Date().toISOString().slice(0, 10),
  doorGua: '',
  masterGua: '',
  stoveGua: '',
})
const layoutResult = ref<BazhaiLayout | null>(null)
const layoutLoading = ref(false)
const layoutReady = computed(() => !!(bazhai.value && fengshuiLayout.doorGua && fengshuiLayout.masterGua && fengshuiLayout.stoveGua))

type FengshuiLayoutPatch = Partial<typeof fengshuiLayout>
function patchFengshuiLayout(patch: FengshuiLayoutPatch) {
  Object.assign(fengshuiLayout, patch)
  if (patch.zuoShan !== undefined || patch.periodDate !== undefined) void loadXuankong()
  if (patch.doorGua !== undefined || patch.masterGua !== undefined || patch.stoveGua !== undefined) void loadLayout()
}

async function loadXuankong() {
  xuankongLoading.value = true
  try {
    xuankong.value = await xuankongChartFor(fengshuiLayout.periodDate, fengshuiLayout.zuoShan)
  } catch {
    xuankong.value = null
  } finally {
    xuankongLoading.value = false
  }
}

async function loadLayout() {
  const chart = bazhai.value
  if (!layoutReady.value || !chart) return
  layoutLoading.value = true
  try {
    layoutResult.value = await bazhaiLayoutFor(chart.ming_gua.gua.name, fengshuiLayout.doorGua, fengshuiLayout.masterGua, fengshuiLayout.stoveGua)
  } catch {
    layoutResult.value = null
  } finally {
    layoutLoading.value = false
  }
}

const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

const formRef = ref<FormInstance>()
// 档案数据问题（canonical 信息缺失/出生年份越界）属字段级错误，显示在档案选择旁。
const rules = computed<FormRules>(() => ({
  profileId: [{
    validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
      const p = profiles.list.find(item => item.id === value)
      if (!value || !p) return callback(new Error(t('chat.selectBirthProfileFirst')))
      if (!p.birth_date) return callback(new Error(t('profiles.chart.errors.missingTime')))
      const birthYear = parseInt(p.birth_date.slice(0, 4), 10)
      if (isNaN(birthYear) || birthYear < 1900 || birthYear > 2100) {
        return callback(new Error(t('services.fengshui.invalidBirthYear')))
      }
      callback()
    },
    trigger: 'change',
  }],
}))

async function onProfileChange() {
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) {
    bazhai.value = null
    error.value = ''
    return
  }
  void loadPreview()
}

// URL 带回档案时（分享/刷新）自动出盘。
onMounted(() => {
  if (profileId.value) void onProfileChange()
})

async function loadPreview() {
  const p = profile.value
  if (!p) return
  loading.value = true
  error.value = ''
  bazhai.value = null
  layoutResult.value = null
  try {
    if (!p.birth_date) {
      error.value = t('profiles.chart.errors.missingTime')
      return
    }
    const birthYear = parseInt(p.birth_date.slice(0, 4), 10)
    bazhai.value = await bazhaiChartFor(birthYear, p.gender)
    void loadXuankong()
  } catch (err) {
    error.value = toUserMessage(err, t('services.fengshui.failed'))
  } finally {
    loading.value = false
  }
}

function saveRecord() {
  const p = profile.value
  if (!bazhai.value || !p) return
  void saveToolRecord(t('records.titleBazhai', { nickname: p.nickname }), { profile_id: p.id })
}

function openRecord(record: ToolRecord) {
  profileId.value = String(record.input.profile_id ?? '')
  void onProfileChange()
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = t('services.fengshui.intent')
  void openServiceChat(p.id, 'fengshui', message => { error.value = message })
}
</script>

<style scoped>
.fx-controls { display: flex; flex-wrap: wrap; gap: var(--space-3); margin-bottom: var(--space-3); }
.fx-select { width: 9rem; }
</style>
