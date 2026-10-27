<template>
  <el-card
    class="app-card chart-preview"
    shadow="never"
  >
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">
          {{ $t('profiles.chart.title') }}
        </h2>
        <div class="app-card__actions">
          <el-button
            size="small"
            type="primary"
            @click="refresh"
          >
            {{ $t('profiles.chart.compute') }}
          </el-button>
        </div>
      </div>
    </template>

    <p
      v-if="correction"
      class="correction"
    >
      {{ correction }}
    </p>

    <p
      v-if="inputError"
      class="hint"
    >
      {{ inputError }}
    </p>

    <el-tabs
      v-if="!inputError"
      v-model="tab"
    >
      <el-tab-pane
        :label="$t('profiles.chart.bazi')"
        name="bazi"
      >
        <EngineQueryPanel
          :key="`${profile.id}:bazi`"
          resource="natal.bazi"
          :input="chartInput"
          :load="loadBaziNatal"
          :enabled="active"
          :fallback="$t('profiles.chart.stageFailed', { stage: $t('profiles.chart.stage.bazi_chart') })"
          :refresh-token="refreshToken"
        >
          <template #default="{ result }">
            <div class="chart-year-picker chart-year-picker--inline">
              <span class="chart-year-label">{{ $t('profiles.chart.liunianYear') }}</span>
              <el-input-number
                v-model="baziLiuNianYear"
                :min="1900"
                :max="2100"
                size="small"
              />
            </div>
            <template v-if="result.baziDetail">
              <BaziFullChartPanel :result="result.baziDetail">
                <template #before-yongshen>
                  <EngineQueryPanel
                    resource="natal.bazi.liunian"
                    :input="{ chart: result.bazi, year: baziLiuNianYear }"
                    :load="loadBaziLiunian"
                    :enabled="!!result.bazi"
                    :fallback="$t('profiles.chart.stageFailed', { stage: $t('profiles.chart.stage.bazi_liunian') })"
                    :refresh-token="refreshToken"
                  >
                    <template #default="{ result: annual }">
                      <BaziLiuNianPanel :result="annual" />
                    </template>
                  </EngineQueryPanel>
                </template>
              </BaziFullChartPanel>
            </template>
            <template v-else>
              <BaziChartPanel :result="result.bazi" />
              <EngineQueryPanel
                resource="natal.bazi.liunian"
                :input="{ chart: result.bazi, year: baziLiuNianYear }"
                :load="loadBaziLiunian"
                :enabled="!!result.bazi"
                :fallback="$t('profiles.chart.stageFailed', { stage: $t('profiles.chart.stage.bazi_liunian') })"
                :refresh-token="refreshToken"
              >
                <template #default="{ result: annual }">
                  <BaziLiuNianPanel :result="annual" />
                </template>
              </EngineQueryPanel>
            </template>
          </template>
        </EngineQueryPanel>
      </el-tab-pane>

      <el-tab-pane
        :label="$t('profiles.chart.ziwei')"
        name="ziwei"
      >
        <EngineQueryPanel
          :key="`${profile.id}:ziwei`"
          resource="natal.ziwei"
          :input="chartInput"
          :load="loadZiweiNatal"
          :enabled="active"
          :fallback="$t('profiles.chart.stageFailed', { stage: $t('profiles.chart.stage.ziwei_chart') })"
          :refresh-token="refreshToken"
        >
          <template #default="{ result }">
            <ZiweiChartPanel :result="result.ziwei" />
            <ZiweiDaXianPanel
              v-if="result.ziweiDaXian.length"
              :result="result.ziweiDaXian"
            />
            <div class="chart-year-picker">
              <el-input-number
                v-model="liuNianYear"
                :min="1900"
                :max="2100"
                size="small"
              />
            </div>
            <EngineQueryPanel
              resource="natal.ziwei.liunian"
              :input="{ base: result.ziweiBase, year: liuNianYear }"
              :load="loadZiweiLiunian"
              :enabled="tab === 'ziwei' && !!result.ziweiBase"
              :fallback="$t('profiles.chart.stageFailed', { stage: $t('profiles.chart.stage.ziwei_liunian') })"
              :refresh-token="refreshToken"
            >
              <template #default="{ result: annual }">
                <ZiweiLiuNianPanel :result="annual" />
              </template>
            </EngineQueryPanel>
          </template>
        </EngineQueryPanel>
      </el-tab-pane>

      <el-tab-pane
        :label="$t('profiles.chart.bazhai')"
        name="bazhai"
      >
        <EngineQueryPanel
          :key="`${profile.id}:bazhai`"
          resource="natal.bazhai"
          :input="bazhaiInput"
          :load="loadBazhai"
          :enabled="active"
          :fallback="$t('services.fengshui.failed')"
          :refresh-token="refreshToken"
        >
          <template #empty>
            <p
              v-if="bazhaiEmptyMessage"
              class="hint"
            >
              {{ bazhaiEmptyMessage }}
            </p>
          </template>
          <template #default="{ result }">
            <BazhaiChartPanel :result="result" />
          </template>
        </EngineQueryPanel>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup lang="ts">
// 档案详情的一屏三盘编排器（App 专属）：只负责资源划分、依赖与布局；
// 每个资源的取数、错误与重试由 EngineQueryPanel 统一承担。
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { BirthProfile } from '@/api/birth-profiles'
import type { ChartInput } from '@shared/domain/time'
import {
  baziLiunianFor,
  baziNatalForChartInput,
  ziweiLiunianFor,
  ziweiNatalForChartInput,
  type BaziLiuNianOutput,
  type BaziNatal,
  type ZiweiChart,
  type ZiweiLiuNianData,
  type ZiweiNatal,
} from '@shared/domain/natal'
import { bazhaiChartFor, type BazhaiChart } from '@shared/domain/fengshui'
import { birthYearFromDate, chartInputFromProfileBirth } from '@shared/domain/birth'
import { formatLongitude, type PlaceLocale } from '@shared/utils/placeDisplay'
import EngineQueryPanel from '@shared/components/engine/EngineQueryPanel.vue'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import ZiweiDaXianPanel from '@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'
import ZiweiLiuNianPanel from '@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'

const props = defineProps<{ profile: BirthProfile; auto?: boolean }>()
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

// tab 状态同步到 URL（?tab=ziwei 可分享/刷新保留）；非法值回落八字。
const TAB_NAMES = ['bazi', 'ziwei', 'bazhai']
const initialTab = String(route.query.tab ?? '')
const tab = ref(TAB_NAMES.includes(initialTab) ? initialTab : 'bazi')
const baziLiuNianYear = ref(new Date().getFullYear())
const liuNianYear = ref(new Date().getFullYear())
const refreshToken = ref(0)
const started = ref(props.auto === true)

const inputError = computed(() => {
  const birth = props.profile.birth
  if (!birth?.instant || !birth?.location?.timezone
    || birth?.location?.longitude == null || !birth.date) {
    return t('profiles.chart.errors.missingTime')
  }
  return ''
})

const active = computed(() => started.value && !inputError.value)
const chartInput = computed(() =>
  inputError.value ? null : chartInputFromProfileBirth(props.profile),
)
const correction = computed(() => chartInput.value
  ? t('profiles.chart.correctedLongitude', {
      longitude: formatLongitude(chartInput.value.longitude, locale.value as PlaceLocale),
    })
  : '')
const birthYear = computed(() => birthYearFromDate(props.profile.birth.date ?? ''))
const bazhaiInput = computed(() => birthYear.value === null
  ? null
  : { birthYear: birthYear.value, gender: props.profile.birth.gender })
const bazhaiEmptyMessage = computed(() =>
  birthYear.value === null && props.profile.birth.date
    ? t('services.fengshui.invalidBirthYear')
    : '')

function loadBaziNatal(input: ChartInput & { gender: string }): Promise<BaziNatal> {
  return baziNatalForChartInput(input)
}

function loadZiweiNatal(input: ChartInput & { gender: string }): Promise<ZiweiNatal> {
  return ziweiNatalForChartInput(input)
}

function loadBazhai(input: { birthYear: number; gender: string }): Promise<BazhaiChart> {
  return bazhaiChartFor(input.birthYear, input.gender)
}

function loadBaziLiunian(input: { chart: BaziNatal['bazi']; year: number }): Promise<BaziLiuNianOutput> {
  return baziLiunianFor(input.chart, input.year)
}

function loadZiweiLiunian(input: { base: ZiweiChart; year: number }): Promise<ZiweiLiuNianData> {
  return ziweiLiunianFor(input.base, input.year)
}

function refresh() {
  started.value = true
  refreshToken.value += 1
}

watch(tab, value => {
  const query = { ...route.query }
  if (value === 'bazi') delete query.tab
  else query.tab = value
  void router.replace({ query })
})
</script>


<style scoped>
.chart-preview {
  border-radius: var(--radius-lg); margin-bottom: var(--space-5);
  position: relative;
  border-top: 2px solid var(--liki-primary);
}
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: var(--space-1) 0 var(--space-3); }

.correction { margin: 0 0 var(--space-3); font-size: var(--font-size-caption); color: var(--liki-primary); }

</style>
