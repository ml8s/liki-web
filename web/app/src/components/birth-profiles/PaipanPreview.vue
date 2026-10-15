<template>
  <el-card
    class="app-card paipan-preview"
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
      v-if="profile.calendar === 'lunar'"
      class="hint"
    >
      {{ $t('profiles.chart.lunarHint') }}
    </p>

    <template v-else>
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

      <div
        v-if="loading"
        class="pan-skeleton"
      >
        <div
          v-for="i in 6"
          :key="i"
          class="skel"
        />
      </div>

      <template v-if="anyResult">
        <el-tabs v-model="tab">
          <el-tab-pane
            :label="$t('profiles.chart.bazi')"
            name="bazi"
          >
            <div class="pillars">
              <div
                v-for="item in baziPillars"
                :key="item.label"
                :class="['pillar', { self: item.self }]"
              >
                <div class="label">
                  {{ item.label }}
                </div>
                <div class="gl font-serif">
                  {{ item.gan }}{{ item.zhi }}
                </div>
                <div class="sub">
                  {{ item.shiShen }}
                </div>
                <div class="sub nayin">
                  {{ item.naYin }}
                </div>
                <div
                  v-if="item.cangGan"
                  class="sub cang"
                >
                  {{ item.cangGan }}
                </div>
                <span
                  v-if="item.isVoid"
                  class="void-tag"
                >{{ $t('profiles.chart.void') }}</span>
              </div>
            </div>
            <div
              v-if="sanYuanText"
              class="san-yuan"
            >
              <span class="sy-label">{{ $t('profiles.chart.threeYuan') }}</span>
              <span class="sy-body">{{ sanYuanText }}</span>
            </div>
            <div
              v-if="shenShaList.length"
              class="shensha"
            >
              <span class="sy-label">{{ $t('profiles.chart.shenSha') }}</span>
              <span
                v-for="s in shenShaList"
                :key="s"
                class="ss-chip"
              >{{ s }}</span>
            </div>
            <template v-if="daYunSteps.length">
              <div class="sec-title">
                {{ $t('profiles.chart.daYun') }}
              </div>
              <div class="dayun">
                <div
                  v-for="(s, i) in daYunSteps"
                  :key="`${s.name}-${i}`"
                  :class="['dy', { cur: i === currentStepIndex }]"
                >
                  <div class="dy-name font-serif">
                    {{ s.name }}
                  </div>
                  <div class="dy-shen">
                    {{ s.shi_shen }}
                  </div>
                  <div class="dy-year">
                    {{ s.start_year }}–{{ s.end_year }}
                  </div>
                </div>
              </div>
            </template>
            <template v-if="yongShenSummary.length">
              <div class="sec-title">
                {{ $t('profiles.chart.yongShen') }}
              </div>
              <div class="ys">
                <div
                  v-for="row in yongShenSummary"
                  :key="row.label"
                  class="ys-row"
                >
                  <span class="ys-label">{{ row.label }}</span>
                  <span class="ys-body">{{ row.text }}</span>
                </div>
              </div>
            </template>
          </el-tab-pane>

          <el-tab-pane
            :label="$t('profiles.chart.ziwei')"
            name="ziwei"
          >
            <div class="palaces">
              <div
                v-for="g in ziweiGongs"
                :key="g.name"
                :class="['palace', { ming: g.name === '命宫', shen: g.is_shen_gong }]"
              >
                <div class="pname">
                  {{ g.name }}
                </div>
                <div class="pstar">
                  {{ mainStars(g) }}
                </div>
                <div class="pminor">
                  {{ minorStars(g) }}
                </div>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane
            :label="$t('profiles.chart.bazhai')"
            name="bazhai"
          >
            <div
              v-if="bazhai"
              class="mingua-row"
            >
              <div class="mingua font-serif">
                {{ bazhai.ming_gua.gua.name }}
              </div>
              <div>
                <div class="mg-name">
                  {{ bazhai.ming_gua.group }}
                </div>
                <div class="mg-sub">
                  {{ $t('profiles.chart.directions') }}
                </div>
              </div>
            </div>
            <div
              v-if="bazhai"
              class="dirs"
            >
              <div
                v-for="d in bazhaiDirs"
                :key="d.key"
                :class="['dir', { good: d.good }]"
              >
                <div class="dg font-serif">
                  {{ d.gua }}
                </div>
                <div class="dname">
                  {{ $t(`profiles.chart.direction.${d.key}`) }}
                </div>
              </div>
            </div>
            <template v-if="liuNianXing">
              <div class="sec-title">
                {{ $t('profiles.chart.annualStars', { year: liuNianXing.year }) }}
              </div>
              <div class="ln-row">
                {{ $t('profiles.chart.centerStar') }}<strong>{{ liuNianXing.ru_zhong }}</strong>
              </div>
              <div class="fly-grid">
                <div
                  v-for="f in flyStars"
                  :key="f.gong_num"
                  :class="['fly-cell', { center: f.gong_num === 5 }]"
                >
                  <div class="fly-name">
                    {{ f.xing_name }}
                  </div>
                </div>
              </div>
            </template>
          </el-tab-pane>
        </el-tabs>
        <div class="correction">
          {{ correction }}
        </div>
      </template>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { BaziFullChartData } from '@/api/engine'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  engineBazhaiChart,
  engineBaziChart,
  engineBaziFullChart,
  engineZiweiChart,
  engineZiweiFullChart,
  type BazhaiChart,
  type BaziChartData,
  type BaziZhuFull,
  type BaziYongShen,
  type ZiweiChart,
} from '@/api/engine'
import { resolvePaipanContext } from '@/utils/paipan'
import { BAZHAI_AUS, LUOSHU_ORDER } from '@/utils/bazhai'
import type { BirthProfile } from '@/api/birth-profiles'

const props = defineProps<{ profile: BirthProfile; auto?: boolean }>()
const { t } = useI18n()

const loading = ref(false)
const error = ref('')
const correction = ref('')
const tab = ref('bazi')
const baziChart = ref<BaziChartData | null>(null)
const baziFull = ref<BaziFullChartData | null>(null)
const ziwei = ref<ZiweiChart | null>(null)
const bazhai = ref<BazhaiChart | null>(null)
const yongShen = ref<BaziYongShen | null>(null)

const anyResult = computed(() => !!(baziChart.value || ziwei.value || bazhai.value))

function zhuText(p?: { gan: string; zhi: string }): string {
  return p ? `${p.gan}${p.zhi}` : ''
}

const baziPillars = computed(() => {
  const src = baziFull.value ?? (baziChart.value as unknown as { nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull } | null)
  if (!src) return []
  const map = (key: 'year' | 'month' | 'day' | 'hour', p: BaziZhuFull) => {
    const cang = p.cang_gan ? [p.cang_gan.main, p.cang_gan.mid, p.cang_gan.minor].filter(Boolean).join(' ') : ''
    return {
      label: t(`profiles.chart.pillar.${key}`),
      self: key === 'day',
      gan: p.gan,
      zhi: p.zhi,
      naYin: p.na_yin ?? '',
      shiShen: p.shi_shens?.[0]?.shi_shen ?? '',
      cangGan: cang ? t('profiles.chart.hiddenStems', { stems: cang }) : '',
      isVoid: !!p.is_void,
    }
  }
  return [
    map('year', src.nian),
    map('month', src.yue),
    map('day', src.ri),
    map('hour', src.shi),
  ]
})

const sanYuanText = computed(() => {
  const sy = baziFull.value?.san_yuan
  if (!sy) return ''
  return t('profiles.chart.sanYuanValue', {
    taiYuan: zhuText(sy.tai_yuan),
    mingGong: zhuText(sy.ming_gong),
    shenGong: zhuText(sy.shen_gong),
  })
})

const shenShaList = computed(() => {
  const src = baziFull.value
  if (!src) return []
  const names = new Set<string>()
  for (const p of [src.nian, src.yue, src.ri, src.shi]) {
    for (const s of p.shen_sha ?? []) {
      if (s.name) names.add(s.name)
    }
  }
  return [...names]
})

const daYunSteps = computed(() => baziChart.value?.da_yun?.steps ?? [])
const currentStepIndex = computed(() => baziChart.value?.da_yun?.current_step_index ?? -1)

const yongShenSummary = computed(() => {
  // 从 baziFull.value 获取 full 对象（由 engineBaziFullChart 返回）
  const full = baziFull.value
  if (!full) return []
  const { fu_yi, tiao_hou, ge_ju } = full // full 即 engineBaziFullChart 返回的对象
  const rows: Array<{ label: string; text: string }> = []

  // 扶抑
  if (fu_yi?.qiangruo) {
    rows.push({
      label: t('profiles.chart.fuYi'),
      text: t('profiles.chart.favorAvoid', { value: fu_yi.qiangruo, favorable: fu_yi.xi ?? '—', unfavorable: fu_yi.ji ?? '—' })
    })
  }
  // 调候
  if (tiao_hou?.yong) {
    rows.push({ label: t('profiles.chart.tiaoHou'), text: t('profiles.chart.useFavor', { value: tiao_hou.yong, favorable: tiao_hou.xi ?? '—' }) })
  }
  // 格局
  if (ge_ju?.ge_ju) {
    rows.push({
      label: t('profiles.chart.geJu'),
      text: t('profiles.chart.patternValue', {
        value: `${ge_ju.ge_ju}${ge_ju.yong_fa ? ` · ${ge_ju.yong_fa}` : ''}`,
        favorable: ge_ju.xi ?? '—',
        unfavorable: ge_ju.ji ?? '—',
      })
    })
  }
  return rows
})

const ziweiGongs = computed(() => ziwei.value?.gong_wei ?? [])
function mainStars(g: { xing_yao: Array<{ name: string; is_zhu_xing?: boolean; liang_du?: string }> }) {
  return g.xing_yao.filter(s => s.is_zhu_xing).map(s => (s.liang_du ? `${s.name}${s.liang_du}` : s.name)).join(' ')
}
function minorStars(g: { xing_yao: Array<{ name: string; is_zhu_xing?: boolean; liang_du?: string }> }) {
  return g.xing_yao.filter(s => !s.is_zhu_xing).map(s => (s.liang_du ? `${s.name}${s.liang_du}` : s.name)).join(' ')
}

const bazhaiDirs = computed(() => {
  if (!bazhai.value) return []
  const dirs = bazhai.value.ba_zhai_dirs
  return Object.entries(dirs).map(([key, values]) => ({
    key, gua: (values ?? [''])[0], good: BAZHAI_AUS.includes(key),
  }))
})

const liuNianXing = computed(() => bazhai.value?.liu_nian_xing ?? null)
const flyStars = computed(() => {
  const gong = liuNianXing.value?.gong_wei
  if (!gong || gong.length !== 9) return []
  // gong_num 1-9：按洛书九宫排 3x3（1下、2右上…按洛书序）
  return LUOSHU_ORDER.map(n => gong.find(g => g.gong_num === n) ?? { gong_num: n, xing_name: '—' })
})

async function compute() {
  const p = props.profile
  error.value = ''
  correction.value = ''
  baziChart.value = null
  baziFull.value = null
  yongShen.value = null
  ziwei.value = null
  bazhai.value = null
  if (p.calendar === 'lunar') return
  if (p.birth_time_known === 'unknown') {
    error.value = t('profiles.chart.errors.unknownTime')
    return
  }
  if (!p.birth_time) {
    error.value = t('profiles.chart.errors.missingTime')
    return
  }
  loading.value = true
  try {
    const ctx = await resolvePaipanContext(p)
    if (ctx.corrected) correction.value = t('profiles.chart.corrected')
    else if (p.birth_time_known === 'known' && !p.birth_place) correction.value = t('profiles.chart.notCorrectedNoPlace')
    else if (p.birth_place) correction.value = t('profiles.chart.approximateWithPlace')
    else correction.value = t('profiles.chart.approximate')

    baziChart.value = await engineBaziChart(ctx.solar, p.gender)
    const full = await engineBaziFullChart(baziChart.value)
    baziFull.value = full
    yongShen.value = full.yong_shen ?? null
    const zwChart = await engineZiweiChart(ctx.lunar, p.gender)
    ziwei.value = await engineZiweiFullChart(zwChart)
    const birthYear = baziChart.value.birth_year ?? parseInt(p.birth_date.slice(0, 4), 10)
    bazhai.value = await engineBazhaiChart(birthYear, p.gender)
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.chart.failed'))
  } finally {
    loading.value = false
  }
}

// 自动排盘：进入档案详情（或切换档案）即出盘，免手动点「排盘」。
watch(() => props.profile?.id, () => {
  if (props.auto && props.profile) void compute()
}, { immediate: true })
</script>

<style scoped>
.paipan-preview {
  border-radius: 12px; margin-bottom: 20px;
  position: relative;
  border-top: 2px solid var(--liki-primary);
}
.hint { color: var(--liki-text-2); font-size: 13px; margin: 4px 0 12px; }
.pan-skeleton .skel {
  height: 14px;
  margin-bottom: 10px;
  border-radius: 6px;
  background: var(--liki-line);
  animation: skeleton-pulse 1.2s var(--liki-ease) infinite;
}

@keyframes skeleton-pulse {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: 8px; overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: 16px 8px; text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.pillar .label { font-size: 12px; color: var(--liki-text-3); margin-bottom: 8px; letter-spacing: 0.08em; }
.pillar .gl { font-size: 25px; font-weight: 700; letter-spacing: 0.02em; }
.pillar.self .gl { color: var(--liki-primary); }
.pillar .sub { font-size: 12px; color: var(--liki-text-2); margin-top: 4px; }
.pillar .nayin { color: var(--liki-text-3); }
.pillar .cang { color: var(--liki-text-3); font-size: 11px; }
.void-tag { display: inline-block; margin-top: 6px; padding: 1px 8px; border-radius: 999px; background: var(--liki-primary-tint); color: var(--liki-primary); font-size: 11px; }

.san-yuan, .shensha { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 12px; font-size: 13px; }
.sy-label { color: var(--liki-text-3); flex: 0 0 2em; }
.sy-body { color: var(--liki-ink); }
.shensha { gap: 6px; }
.ss-chip { padding: 2px 10px; border-radius: 999px; border: 1px solid var(--liki-line); background: var(--liki-surface); color: var(--liki-text-2); font-size: 12px; }

.ln-row { font-size: 13px; color: var(--liki-text-2); margin-bottom: 10px; }
.fly-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; width: min(320px, 100%); }
.fly-cell { background: var(--liki-surface); padding: 10px 4px; text-align: center; }
.fly-cell.center { background: var(--liki-primary-tint); }
.fly-name { font-size: 12px; color: var(--liki-text-2); }

.sec-title { font-size: 13px; font-weight: 600; margin: 16px 0 8px; color: var(--liki-ink); }
.dayun { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 6px; }
.dy { border: 1px solid var(--liki-line); border-radius: 6px; padding: 8px 4px; text-align: center; }
.dy.cur { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.dy-name { font-size: 15px; font-weight: 700; }
.dy-shen { font-size: 11px; color: var(--liki-text-2); margin-top: 2px; }
.dy-year { font-size: 11px; color: var(--liki-text-3); margin-top: 2px; }
.ys { border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; }
.ys-row { display: flex; gap: 10px; padding: 8px 12px; font-size: 13px; }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 3em; }
.ys-body { color: var(--liki-ink); }

.palaces {
  display: grid; grid-template-columns: repeat(6, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: 6px; overflow: hidden;
}
.palace { background: var(--liki-surface); padding: 10px 4px; text-align: center; min-height: 68px; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.palace.ming { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.palace.shen { background: var(--liki-primary-tint); }
.palace .pname { font-size: 13px; font-weight: 600; }
.palace.ming .pname, .palace.ming .pstar { color: var(--liki-primary); }
.palace .pstar { font-size: 12px; color: var(--liki-primary); margin-top: 3px; }
.palace .pminor { font-size: 11px; color: var(--liki-text-3); margin-top: 2px; }

.mingua-row { display: flex; align-items: center; gap: 16px; margin-bottom: 14px; }
.mingua { display: grid; place-items: center; width: 60px; height: 60px; border-radius: 50%; border: 2px solid var(--liki-primary); font-size: 26px; color: var(--liki-primary); background: var(--liki-primary-tint); }
.mg-name { font-size: 15px; font-weight: 600; }
.mg-sub { color: var(--liki-text-2); font-size: 12px; margin-top: 2px; }
.dirs { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.dir { border: 1px solid var(--liki-line); border-radius: 6px; padding: 10px; text-align: center; }
.dir .dg { font-size: 18px; color: var(--liki-primary); }
.dir.good { border-color: var(--color-success-border); background: var(--color-success-surface); }
.dir.good .dg { color: var(--el-color-success); }
.dir .dname { font-size: 12px; color: var(--liki-text-2); margin-top: 3px; }

.correction { margin-top: 12px; font-size: 12px; color: var(--liki-primary); }
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
  .palaces { grid-template-columns: repeat(3, 1fr); }
  .dirs { grid-template-columns: repeat(2, 1fr); }
}
</style>
