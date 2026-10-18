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
      <el-select
        v-model="profileId"
        :aria-label="$t('chat.selectBirthProfile')"
        :placeholder="$t('chat.selectBirthProfile')"
        class="w-full"
        @change="profileId && loadPreview()"
      >
        <el-option
          v-for="p in profiles.list"
          :key="p.id"
          :value="p.id"
          :label="p.nickname"
        />
      </el-select>
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
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.fengshui.lifeGua') }}
        </h2>
      </template>
      {{ $t('profiles.chart.compute') }}…
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

      <AppConsultationCard
        :title="$t('services.fengshui.consultationTitle')"
        :description="$t('services.fengshui.consultationBody')"
        :locked-description="$t('services.fengshui.lockedBody')"
        :purchased="entitlements.purchased"
        :action-label="$t('services.fengshui.consultationTitle')"
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
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { engineBazhaiChart, type BazhaiChart } from '@/api/engine'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import MingGuaSection from '@/components/bazhai/MingGuaSection.vue'
import LiuNianFlySection from '@/components/bazhai/LiuNianFlySection.vue'

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

const profileId = ref('')
const bazhai = ref<BazhaiChart | null>(null)
const loading = ref(false)
const error = ref('')

const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

async function loadPreview() {
  const p = profile.value
  if (!p) return
  loading.value = true
  error.value = ''
  bazhai.value = null
  try {
    if (p.calendar === 'lunar') throw new Error(t('profiles.chart.lunarHint'))
    const birthYear = parseInt(p.birth_date.slice(0, 4), 10)
    if (isNaN(birthYear) || birthYear < 1900 || birthYear > 2100) throw new Error(t('services.fengshui.invalidBirthYear'))
    bazhai.value = await engineBazhaiChart(birthYear, p.gender)
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
  void loadPreview()
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = t('services.fengshui.intent')
  void openServiceChat(p.id, 'fengshui', message => { error.value = message })
}
</script>
