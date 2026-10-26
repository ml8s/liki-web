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
            :loading="loading"
            @click="compute"
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

    <div>
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        :closable="false"
        show-icon
      />
      <el-alert
        v-if="stageErrors.length"
        :title="$t('profiles.chart.partialFailed')"
        :description="stageErrorsText"
        type="warning"
        :closable="false"
        show-icon
      />
      <p
        v-if="!anyResult && !loading"
        class="hint"
      >
        {{ $t('profiles.chart.empty') }}
      </p>

      <el-skeleton
        v-if="loading"
        :rows="6"
        animated
        aria-busy="true"
      />

      <template v-if="anyResult">
        <el-tabs v-model="tab">
          <el-tab-pane
            :label="$t('profiles.chart.bazi')"
            name="bazi"
          >
            <div class="chart-year-picker chart-year-picker--inline">
              <span class="chart-year-label">{{ $t('profiles.chart.liunianYear') }}</span>
              <el-input-number
                v-model="baziLiuNianYear"
                :min="1900"
                :max="2100"
                size="small"
                @change="loadBaziLiuNian"
              />
            </div>
            <BaziFullChartPanel
              v-if="baziFull"
              :result="baziFull"
            >
              <template #before-yongshen>
                <BaziLiuNianPanel
                  v-if="baziLiuNian"
                  :result="baziLiuNian"
                />
              </template>
            </BaziFullChartPanel>
            <template v-else-if="baziChart">
              <BaziChartPanel
                :result="baziChart"
              />
              <BaziLiuNianPanel
                v-if="baziLiuNian"
                :result="baziLiuNian"
              />
            </template>
          </el-tab-pane>

          <el-tab-pane
            :label="$t('profiles.chart.ziwei')"
            name="ziwei"
          >
            <template v-if="ziwei">
              <ZiweiChartPanel :result="ziwei" />
              <ZiweiDaXianPanel
                v-if="daXianRaw.length"
                :result="daXianRaw"
              />
              <div class="chart-year-picker">
                <el-input-number
                  v-model="liuNianYear"
                  :min="1900"
                  :max="2100"
                  size="small"
                  @change="loadLiuNian"
                />
              </div>
              <ZiweiLiuNianPanel
                v-if="liuNian"
                :result="liuNian"
              />
            </template>
          </el-tab-pane>
          <el-tab-pane
            :label="$t('profiles.chart.bazhai')"
            name="bazhai"
          >
            <template v-if="bazhai">
              <BazhaiChartPanel :result="bazhai" />
            </template>
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>
  </el-card>
</template>

<script setup lang="ts">
// 档案详情的一屏三盘编排器（App 专属）：数据链 + tab 外壳；
// 盘面组合来自标准 engine result catalog，年选交互保留在宿主。
import { toUserMessage } from '@/utils/errors'
import {
  baziLiunianFor,
  natalChartForChartInput,
  ziweiLiunianFor,
  type BaziChartData,
  type BaziFullChartData,
  type BaziLiuNianOutput,
  type ZiweiChart,
  type ZiweiDaXianItem,
  type ZiweiLiuNianData,
} from '@shared/domain/natal'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { bazhaiChartFor, type BazhaiChart } from '@shared/domain/fengshui'
import { NatalStageError } from '@shared/domain/natal'
import type { BirthProfile } from '@/api/birth-profiles'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import ZiweiDaXianPanel from '@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'
import ZiweiLiuNianPanel from '@shared/components/engine/ziwei/ZiweiLiuNianPanel.vue'
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import { formatLongitude, type PlaceLocale } from '@shared/utils/placeDisplay'

const props = defineProps<{ profile: BirthProfile; auto?: boolean }>()
const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const loading = ref(false)
const error = ref('')
const stageErrors = ref<Array<{ stage: string; detail: string }>>([])
const correction = ref('')
// tab 状态同步到 URL（?tab=ziwei 可分享/刷新保留）；非法值回落八字。
const TAB_NAMES = ['bazi', 'ziwei', 'bazhai']
const initialTab = String(route.query.tab ?? '')
const tab = ref(TAB_NAMES.includes(initialTab) ? initialTab : 'bazi')
const baziChart = ref<BaziChartData | null>(null)
const baziFull = ref<BaziFullChartData | null>(null)
const baziLiuNian = ref<BaziLiuNianOutput | null>(null)
const baziLiuNianYear = ref(new Date().getFullYear())
const ziwei = ref<ZiweiChart | null>(null)
const ziweiBase = ref<ZiweiChart | null>(null)
const bazhai = ref<BazhaiChart | null>(null)
const daXianRaw = ref<ZiweiDaXianItem[]>([])
const liuNianYear = ref(new Date().getFullYear())
const liuNian = ref<ZiweiLiuNianData | null>(null)

const anyResult = computed(() => !!(baziChart.value || ziwei.value || bazhai.value))

function stageLabel(stage: string): string {
  return t(`profiles.chart.stage.${stage}`)
}

function errorDetail(error: unknown): string {
  if (error instanceof NatalStageError) {
    return error.cause instanceof Error ? error.cause.message : String(error.cause)
  }
  return error instanceof Error ? error.message : ''
}

function setStageError(stage: string, error: unknown) {
  const detail = errorDetail(error)
  const index = stageErrors.value.findIndex(item => item.stage === stage)
  const item = { stage, detail }
  if (index >= 0) stageErrors.value[index] = item
  else stageErrors.value.push(item)
}

function clearStageError(stage: string) {
  stageErrors.value = stageErrors.value.filter(item => item.stage !== stage)
}

const stageErrorsText = computed(() =>
  stageErrors.value
    .map(item => `${stageLabel(item.stage)}（${item.stage}）${item.detail ? `：${item.detail}` : ''}`)
    .join('；'),
)

function fatalStageMessage(error: unknown): string {
  if (!(error instanceof NatalStageError)) {
    return toUserMessage(error, t('profiles.chart.failed'))
  }
  const detail = errorDetail(error)
  return `${t('profiles.chart.stageFailed', { stage: stageLabel(error.stage) })}（${error.stage}）${detail ? `：${detail}` : ''}`
}

async function loadLiuNian() {
  if (!ziweiBase.value) return
  try {
    liuNian.value = await ziweiLiunianFor(ziweiBase.value, liuNianYear.value)
    clearStageError('ziwei_liunian')
  } catch (err) {
    // Optional annual view must not replace a successfully rendered natal chart.
    liuNian.value = null
    setStageError('ziwei_liunian', err)
  }
}

// 八字流年盘（bazi.liunian）：失败降级为不显示流年列，不阻塞主盘。
async function loadBaziLiuNian() {
  if (!baziChart.value) return
  try {
    baziLiuNian.value = await baziLiunianFor(baziChart.value, baziLiuNianYear.value)
    clearStageError('bazi_liunian')
  } catch (err) {
    baziLiuNian.value = null
    setStageError('bazi_liunian', err)
  }
}

async function compute() {
  const p = props.profile
  error.value = ''
  stageErrors.value = []
  correction.value = ''
  baziChart.value = null
  baziFull.value = null
  baziLiuNian.value = null
  ziwei.value = null
  ziweiBase.value = null
  bazhai.value = null
  daXianRaw.value = []
  liuNian.value = null
  if (!p.birth?.instant || !p.birth?.location?.timezone || p.birth?.location?.longitude == null) {
    error.value = t('profiles.chart.errors.missingTime')
    return
  }
  if (!p.birth.date) {
    error.value = t('profiles.chart.errors.missingTime')
    return
  }
  loading.value = true
  try {
    const ctx = {
      instant: p.birth.instant,
      timezone: p.birth.location.timezone,
      longitude: p.birth.location.longitude,
    }
    correction.value = t('profiles.chart.correctedLongitude', {
      longitude: formatLongitude(Number(ctx.longitude), locale.value as PlaceLocale),
    })

    const chart = await natalChartForChartInput({ ...ctx, gender: p.birth.gender })
    baziChart.value = chart.bazi
    baziFull.value = chart.baziDetail
    ziwei.value = chart.ziwei
    ziweiBase.value = chart.ziweiBase
    daXianRaw.value = chart.ziweiDaXian
    await loadBaziLiuNian()
    const birthYear = parseInt(p.birth.date.slice(0, 4), 10)
    try {
      bazhai.value = await bazhaiChartFor(birthYear, p.birth.gender)
      clearStageError('bazhai_chart')
    } catch (err) {
      // Bazhai is an optional extra panel; keep the main chart usable.
      bazhai.value = null
      setStageError('bazhai_chart', err)
    }
    if (tab.value === 'ziwei') void loadLiuNian()
  } catch (err) {
    error.value = fatalStageMessage(err)
  } finally {
    loading.value = false
  }
}

// 流年懒加载（切到紫微 tab 时按当前年份查询）+ tab 状态写入 URL。
watch(tab, value => {
  if (value === 'ziwei') void loadLiuNian()
  const query = { ...route.query }
  if (value === 'bazi') delete query.tab
  else query.tab = value
  void router.replace({ query })
})

// 自动排盘：进入档案详情（或切换档案）即出盘，免手动点「排盘」。
watch(() => props.profile?.id, () => {
  if (props.auto && props.profile) void compute()
}, { immediate: true })
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
