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
            <div class="bazi-ln-pick">
              <span class="bazi-ln-label">{{ $t('profiles.chart.liunianYear') }}</span>
              <el-input-number
                v-model="baziLiuNianYear"
                :min="1900"
                :max="2100"
                size="small"
                @change="loadBaziLiuNian"
              />
            </div>
            <BaziPanel
              :chart="baziChart"
              :full="baziFull"
              :liunian="baziLiuNian"
            />
          </el-tab-pane>

          <el-tab-pane
            :label="$t('profiles.chart.ziwei')"
            name="ziwei"
          >
            <template v-if="ziwei">
              <ZiweiPanel
                :chart="ziwei"
                :da-xian="daXianRaw"
              />
              <div class="zw-section zw-liunian">
                <div class="sec-title">
                  {{ $t('profiles.chart.liunian') }}
                </div>
                <div class="ln-pick">
                  <el-input-number
                    v-model="liuNianYear"
                    :min="1900"
                    :max="2100"
                    size="small"
                    @change="loadLiuNian"
                  />
                </div>
                <template v-if="liuNian">
                  <div class="zwc-sihua ln-sihua">
                    <span
                      v-for="(hua, star) in liuNian.si_hua ?? {}"
                      :key="star"
                      class="zw-sihua-chip"
                    >{{ star }}<i
                      class="zw-si"
                      :data-hua="hua"
                    >{{ hua }}</i>{{ liuNian.si_hua_gong?.[star] ? ` ${liuNian.si_hua_gong[star]}` : '' }}</span>
                  </div>
                  <div class="ln-grid">
                    <div
                      v-for="g in liuNianPlaced"
                      :key="`${g.name}-${g.zhi}`"
                      :style="g.style"
                      :class="['ln-cell', { ming: g.ming }]"
                    >
                      <div class="ln-pname">
                        {{ g.name }}
                      </div>
                      <div
                        v-if="g.stars.length"
                        class="ln-stars"
                      >
                        {{ g.stars.join(' ') }}
                      </div>
                    </div>
                    <div class="ln-center">
                      <div class="lnc-title">
                        {{ liuNianYear }}
                      </div>
                      <div class="lnc-row">
                        {{ $t('profiles.chart.liuNianPalace', { name: liuNian.ming_gong_name ?? liuNian.ming_gong }) }}
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </template>
          </el-tab-pane>

          <el-tab-pane
            :label="$t('profiles.chart.bazhai')"
            name="bazhai"
          >
            <template v-if="bazhai">
              <MingGuaSection :bazhai="bazhai" />
              <LiuNianFlySection
                v-if="bazhai.liu_nian_xing"
                :liu-nian="bazhai.liu_nian_xing"
              />
            </template>
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 档案详情的一屏三盘编排器（App 专属）：数据链 + tab 外壳；
// 三块盘面来自共享层（web/shared/components/charts），营销站 island 单独复用。
import { toUserMessage } from '@/utils/errors'
import {
  baziLiunianFor,
  natalChartForChartInput,
  ziweiLiunianFor,
  type BaziChartData,
  type BaziFullChartData,
  type BaziLiuNianOutput,
  type BaziYongShen,
  type ZiweiChart,
  type ZiweiDaXianItem,
  type ZiweiLiuNianData,
} from '@shared/domain/natal'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { bazhaiChartFor, type BazhaiChart } from '@shared/domain/fengshui'
import type { BirthProfile } from '@/api/birth-profiles'
import BaziPanel from '@shared/components/charts/BaziPanel.vue'
import ZiweiPanel from '@shared/components/charts/ZiweiPanel.vue'
import MingGuaSection from '@shared/components/charts/MingGuaSection.vue'
import LiuNianFlySection from '@shared/components/charts/LiuNianFlySection.vue'

const props = defineProps<{ profile: BirthProfile; auto?: boolean }>()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const loading = ref(false)
const error = ref('')
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
const bazhai = ref<BazhaiChart | null>(null)
const yongShen = ref<BaziYongShen | null>(null)
const daXianRaw = ref<ZiweiDaXianItem[]>([])
const liuNianYear = ref(new Date().getFullYear())
const liuNian = ref<ZiweiLiuNianData | null>(null)

const anyResult = computed(() => !!(baziChart.value || ziwei.value || bazhai.value))

const ZHI_POS: Record<string, string> = {
  巳: 'grid-row:1;grid-column:1',
  午: 'grid-row:1;grid-column:2',
  未: 'grid-row:1;grid-column:3',
  申: 'grid-row:1;grid-column:4',
  辰: 'grid-row:2;grid-column:1',
  酉: 'grid-row:2;grid-column:4',
  卯: 'grid-row:3;grid-column:1',
  戌: 'grid-row:3;grid-column:4',
  寅: 'grid-row:4;grid-column:1',
  丑: 'grid-row:4;grid-column:2',
  子: 'grid-row:4;grid-column:3',
  亥: 'grid-row:4;grid-column:4',
}

const liuNianPlaced = computed(() => {
  const data = liuNian.value
  if (!data?.gong_wei) return []
  return data.gong_wei.map(g => ({
    name: g.name,
    zhi: g.zhi,
    ming: !!g.is_liu_ming,
    stars: g.xing_yao ?? [],
    style: ZHI_POS[g.zhi ?? ''] ?? '',
  }))
})

async function loadLiuNian() {
  if (!ziwei.value) return
  try {
    liuNian.value = await ziweiLiunianFor(ziwei.value, liuNianYear.value)
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.chart.failed'))
  }
}

// 八字流年盘（bazi.liunian）：失败降级为不显示流年列，不阻塞主盘。
async function loadBaziLiuNian() {
  if (!baziChart.value) return
  try {
    baziLiuNian.value = await baziLiunianFor(baziChart.value, baziLiuNianYear.value)
  } catch {
    baziLiuNian.value = null
  }
}

async function compute() {
  const p = props.profile
  error.value = ''
  correction.value = ''
  baziChart.value = null
  baziFull.value = null
  baziLiuNian.value = null
  yongShen.value = null
  ziwei.value = null
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
    correction.value = t('profiles.chart.correctedLongitude', { lng: Number(ctx.longitude).toFixed(4) })

    const chart = await natalChartForChartInput({ ...ctx, gender: p.birth.gender })
    baziChart.value = chart.bazi
    baziFull.value = chart.baziDetail
    yongShen.value = chart.baziDetail.yong_shen ?? null
    ziwei.value = chart.ziwei
    daXianRaw.value = chart.ziweiDaXian
    await loadBaziLiuNian()
    const birthYear = parseInt(p.birth.date.slice(0, 4), 10)
    bazhai.value = await bazhaiChartFor(birthYear, p.birth.gender)
    if (tab.value === 'ziwei') void loadLiuNian()
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.chart.failed'))
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

.zw-section { margin-top: var(--space-4); }
.zw-si {
  display: inline-block; width: 14px; height: 14px; line-height: 14px; border-radius: var(--radius-full);
  font-style: normal; font-size: var(--font-size-micro); text-align: center; margin-left: 1px; color: var(--color-white);
}
.zw-si[data-hua="禄"] { background: var(--color-success); }
.zw-si[data-hua="权"] { background: var(--el-color-primary); }
.zw-si[data-hua="科"] { background: var(--el-color-warning); }
.zw-si[data-hua="忌"] { background: var(--el-color-danger); }

.ln-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: var(--liki-line);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-small);
  overflow: hidden;
  grid-template-rows: repeat(4, minmax(60px, auto)); margin-top: var(--space-2);
}
.ln-cell { background: var(--liki-surface); min-width: 0; padding: 4px 5px; }
.ln-cell.ming { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.ln-pname { font-size: var(--font-size-micro); font-weight: 600; color: var(--liki-text-2); white-space: nowrap; }
.ln-cell.ming .ln-pname { color: var(--liki-primary); }
.ln-stars { font-size: var(--font-size-micro); color: var(--liki-primary); line-height: 1.35; margin-top: 2px; }
.ln-center { grid-row: 2 / 4; grid-column: 2 / 4; background: var(--liki-surface-cream); display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 6px; text-align: center; }
.lnc-title { font-size: var(--font-size-body-lg); font-weight: 700; color: var(--liki-primary); }
.lnc-row { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: 2px; }

.ln-pick { margin-bottom: var(--space-2); }
.bazi-ln-pick { display: flex; align-items: center; gap: var(--space-2); margin-bottom: var(--space-2); }
.bazi-ln-label { font-size: var(--font-size-label); color: var(--liki-text-3); }
.ln-sihua { justify-content: flex-start; }

@media (max-width: 640px) {
  .ln-grid { grid-template-rows: repeat(4, minmax(52px, auto)); }
  .ln-cell { padding: 3px; }
}
</style>
