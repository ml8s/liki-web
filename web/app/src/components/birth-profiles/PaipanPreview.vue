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
            <template v-if="ziwei">
              <div class="zw-grid">
                <div
                  v-for="g in ziweiPlaced"
                  :key="g.name"
                  :style="g.style"
                  :class="['zw-palace', { ming: g.ming, shen: g.shen }]"
                  :title="g.tooltip"
                >
                  <div class="zw-head">
                    <span class="zw-pname">{{ g.name }}</span>
                    <span class="zw-gz font-serif">{{ g.gan }}{{ g.zhi }}</span>
                  </div>
                  <div
                    v-if="g.main.length"
                    class="zw-main"
                  >
                    <span
                      v-for="s in g.main"
                      :key="s.name"
                      class="zw-star"
                    >{{ s.name }}{{ s.liang_du }}<i
                      v-if="s.si_hua"
                      class="zw-si"
                      :data-hua="s.si_hua"
                    >{{ s.si_hua }}</i></span>
                  </div>
                  <div
                    v-if="g.minor.length"
                    class="zw-minor"
                  >
                    <span
                      v-for="s in g.minor"
                      :key="s.name"
                      class="zw-star"
                    >{{ s.name }}{{ s.liang_du }}<i
                      v-if="s.si_hua"
                      class="zw-si"
                      :data-hua="s.si_hua"
                    >{{ s.si_hua }}</i></span>
                  </div>
                  <div
                    v-if="g.zaYao.length"
                    class="zw-za"
                  >{{ g.zaYao.join(' ') }}</div>
                  <div class="zw-foot">
                    <span>{{ $t('profiles.chart.changSheng') }} {{ g.changSheng }}</span>
                    <b
                      v-if="g.shen"
                      class="zw-shen"
                    >身</b>
                  </div>
                </div>
                <div class="zw-center">
                  <div class="zwc-title">
                    {{ ziwei.ju_shu }}
                  </div>
                  <div class="zwc-row">
                    {{ $t('profiles.chart.mingZhu') }} {{ ziwei.ming_zhu }} · {{ $t('profiles.chart.shenZhu') }} {{ ziwei.shen_zhu }}
                  </div>
                  <div class="zwc-row">
                    {{ $t('profiles.chart.ziweiPos') }} {{ ziwei.ziwei_pos }} · {{ $t('profiles.chart.shenGong') }} {{ ziwei.shen_gong }}
                  </div>
                  <div class="zwc-row">
                    {{ $t('profiles.chart.ziweiBirth', { nian: `${ziwei.nian_gan}${ziwei.nian_zhi}`, shi: ziwei.shi_zhi }) }}
                  </div>
                  <div class="zwc-sihua">
                    <span
                      v-for="(hua, star) in ziwei.si_hua ?? {}"
                      :key="star"
                      class="zw-sihua-chip"
                    >{{ star }}<i
                      class="zw-si"
                      :data-hua="hua"
                    >{{ hua }}</i></span>
                  </div>
                </div>
              </div>
              <div
                v-if="ziwei.patterns?.length"
                class="zw-section"
              >
                <div class="sec-title">
                  {{ $t('profiles.chart.patterns') }}
                </div>
                <div class="zw-chips">
                  <span
                    v-for="p in ziwei.patterns"
                    :key="p.name"
                    class="zw-chip"
                    :title="p.description"
                  >{{ p.name }}</span>
                </div>
              </div>
              <div
                v-if="ziwei.san_fang?.length"
                class="zw-section"
              >
                <div class="sec-title">
                  {{ $t('profiles.chart.sanFang') }}
                </div>
                <div class="zw-sanfang">
                  <div
                    v-for="s in ziwei.san_fang"
                    :key="s.name"
                    class="zw-sf"
                  >
                    <div class="zw-sf-name">
                      {{ s.name }}
                    </div>
                    <div class="zw-sf-body">
                      {{ s.zhu_xing.join(' ') }}<template v-if="s.fu_xing?.length"> · {{ s.fu_xing.join(' ') }}</template><i
                        v-for="(hua, i) in s.si_hua ?? []"
                        :key="i"
                        class="zw-si"
                        :data-hua="hua"
                      >{{ hua }}</i>
                    </div>
                  </div>
                </div>
              </div>
              <div
                v-if="ziwei.kong_gong?.length"
                class="zw-section"
              >
                <div class="sec-title">
                  {{ $t('profiles.chart.kongJie') }}
                </div>
                <div class="zw-chips">
                  <span
                    v-for="k in ziwei.kong_gong"
                    :key="k.gong_name"
                    class="zw-chip"
                  >{{ $t('profiles.chart.kongJieValue', { gong: k.gong_name, dui: k.dui_gong, stars: k.jie_xing.join(' ') }) }}</span>
                </div>
              </div>
              <div
                v-if="daXian.length"
                class="zw-section zw-daxian"
              >
                <div class="sec-title">
                  {{ $t('profiles.chart.daxian') }}
                </div>
                <div class="dayun">
                  <div
                    v-for="d in daXian"
                    :key="`${d.name}-${d.qi_sui}`"
                    :class="['dy', { cur: d.current }]"
                  >
                    <div class="dy-name">
                      {{ d.name }}
                    </div>
                    <div class="dy-shen">
                      {{ $t('profiles.chart.ageRange', { from: d.qi_sui, to: d.zhi_sui }) }}
                    </div>
                    <div class="dy-year">
                      {{ d.yearRange }}
                    </div>
                  </div>
                </div>
              </div>
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
  engineZiweiDaXian,
  engineZiweiFullChart,
  engineZiweiLiuNian,
  type BazhaiChart,
  type BaziChartData,
  type BaziZhuFull,
  type BaziYongShen,
  type ZiweiChart,
  type ZiweiDaXianItem,
  type ZiweiLiuNianData,
  type ZiweiStarInfo,
} from '@/api/engine'
import { resolvePaipanContext } from '@/utils/paipan'
import type { BirthProfile } from '@/api/birth-profiles'
import MingGuaSection from '@/components/bazhai/MingGuaSection.vue'
import LiuNianFlySection from '@/components/bazhai/LiuNianFlySection.vue'

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
const daXianRaw = ref<ZiweiDaXianItem[]>([])
const liuNianYear = ref(new Date().getFullYear())
const liuNian = ref<ZiweiLiuNianData | null>(null)

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

interface ZiweiPalaceVM {
  name: string
  gan: string
  zhi: string
  ming: boolean
  shen: boolean
  style: string
  main: ZiweiStarInfo[]
  minor: ZiweiStarInfo[]
  zaYao: string[]
  changSheng: string
  tooltip: string
}

const ziweiPlaced = computed<ZiweiPalaceVM[]>(() => {
  const chart = ziwei.value
  if (!chart) return []
  return (chart.gong_wei ?? []).map(g => {
    const ages = (g.ages ?? []).join('、')
    return {
      name: g.name,
      gan: g.gan ?? '',
      zhi: g.zhi ?? '',
      ming: g.name === chart.ming_gong,
      shen: !!g.is_shen_gong,
      style: ZHI_POS[g.zhi ?? ''] ?? '',
      main: (g.xing_yao ?? []).filter(s => s.is_zhu_xing),
      minor: (g.xing_yao ?? []).filter(s => !s.is_zhu_xing),
      zaYao: g.za_yao ?? [],
      changSheng: g.chang_sheng ?? '',
      tooltip: ages ? t('profiles.chart.xiaoXian', { ages }) : '',
    }
  })
})

const daXian = computed(() => {
  const cy = new Date().getFullYear()
  return daXianRaw.value.map(d => ({
    ...d,
    yearRange: d.start_year != null && d.end_year != null ? `${d.start_year}–${d.end_year}` : '',
    current: d.start_year != null && d.end_year != null && cy >= d.start_year && cy <= d.end_year,
  }))
})

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
    liuNian.value = await engineZiweiLiuNian(ziwei.value, liuNianYear.value)
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.chart.failed'))
  }
}

async function compute() {
  const p = props.profile
  error.value = ''
  correction.value = ''
  baziChart.value = null
  baziFull.value = null
  yongShen.value = null
  ziwei.value = null
  bazhai.value = null
  daXianRaw.value = []
  liuNian.value = null
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
    daXianRaw.value = await engineZiweiDaXian(ziwei.value)
    if (tab.value === 'ziwei') void loadLiuNian()
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.chart.failed'))
  } finally {
    loading.value = false
  }
}

// 流年懒加载：切到紫微 tab 时按当前年份查询。
watch(tab, value => {
  if (value === 'ziwei') void loadLiuNian()
})

// 自动排盘：进入档案详情（或切换档案）即出盘，免手动点「排盘」。
watch(() => props.profile?.id, () => {
  if (props.auto && props.profile) void compute()
}, { immediate: true })
</script>

<style scoped>
.paipan-preview {
  border-radius: var(--radius-lg); margin-bottom: 20px;
  position: relative;
  border-top: 2px solid var(--liki-primary);
}
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: 4px 0 12px; }
.pan-skeleton .skel {
  height: 14px;
  margin-bottom: 10px;
  border-radius: var(--radius-small);
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
  border-radius: var(--radius-md); overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: 16px 8px; text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.pillar .label { font-size: var(--font-size-caption); color: var(--liki-text-3); margin-bottom: 8px; letter-spacing: 0.08em; }
.pillar .gl { font-size: 25px; font-weight: 700; letter-spacing: 0.02em; }
.pillar.self .gl { color: var(--liki-primary); }
.pillar .sub { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: 4px; }
.pillar .nayin { color: var(--liki-text-3); }
.pillar .cang { color: var(--liki-text-3); font-size: var(--font-size-micro); }
.void-tag { display: inline-block; margin-top: 6px; padding: 1px 8px; border-radius: var(--radius-full); background: var(--liki-primary-tint); color: var(--liki-primary); font-size: var(--font-size-micro); }

.san-yuan, .shensha { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 12px; font-size: var(--font-size-label); }
.sy-label { color: var(--liki-text-3); flex: 0 0 2em; }
.sy-body { color: var(--liki-ink); }
.shensha { gap: 6px; }
.ss-chip { padding: 2px 10px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); background: var(--liki-surface); color: var(--liki-text-2); font-size: var(--font-size-caption); }

.correction { margin-top: 12px; font-size: var(--font-size-caption); color: var(--liki-primary); }
.sec-title { font-size: var(--font-size-label); font-weight: 600; margin: 16px 0 8px; color: var(--liki-ink); }
.dayun { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 6px; }
.dy { border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: 8px 4px; text-align: center; }
.dy.cur { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.dy-name { font-size: var(--font-size-body-lg); font-weight: 700; }
.dy-shen { font-size: var(--font-size-micro); color: var(--liki-text-2); margin-top: 2px; }
.dy-year { font-size: var(--font-size-micro); color: var(--liki-text-3); margin-top: 2px; }
.ys { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.ys-row { display: flex; gap: 10px; padding: 8px 12px; font-size: var(--font-size-label); }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 3em; }
.ys-body { color: var(--liki-ink); }

.zw-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: repeat(4, minmax(100px, auto));
  gap: 1px;
  background: var(--liki-line);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-small);
  overflow: hidden;
}
.zw-palace { background: var(--liki-surface); padding: 6px 5px; display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.zw-palace.ming { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.zw-palace.ming .zw-pname, .zw-palace.ming .zw-main { color: var(--liki-primary); }
.zw-palace.shen { box-shadow: inset 0 0 0 1px var(--liki-primary-border); }
.zw-head { display: flex; justify-content: space-between; align-items: baseline; gap: 4px; }
.zw-pname { font-size: var(--font-size-label); font-weight: 600; white-space: nowrap; }
.zw-gz { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.zw-main { color: var(--liki-primary); font-size: var(--font-size-caption); line-height: 1.4; }
.zw-minor { color: var(--liki-text-2); font-size: var(--font-size-micro); line-height: 1.4; }
.zw-star { white-space: nowrap; margin-right: 4px; display: inline-block; }
.zw-si {
  display: inline-block; width: 14px; height: 14px; line-height: 14px; border-radius: var(--radius-full);
  font-style: normal; font-size: 10px; text-align: center; margin-left: 1px; color: var(--color-white);
}
.zw-si[data-hua="禄"] { background: var(--el-color-success); }
.zw-si[data-hua="权"] { background: var(--el-color-primary); }
.zw-si[data-hua="科"] { background: var(--el-color-warning); }
.zw-si[data-hua="忌"] { background: var(--el-color-danger); }
.zw-za { font-size: var(--font-size-micro); color: var(--liki-text-3); line-height: 1.35; }
.zw-foot { margin-top: auto; display: flex; justify-content: space-between; align-items: center; gap: 4px; font-size: var(--font-size-micro); color: var(--liki-text-3); }
.zw-shen { color: var(--liki-primary); font-weight: 700; }
.zw-center {
  grid-row: 2 / 4; grid-column: 2 / 4;
  background: var(--liki-surface-cream);
  padding: 10px 12px; display: flex; flex-direction: column; gap: 6px; justify-content: center; text-align: center;
}
.zwc-title { font-size: var(--font-size-body-lg); font-weight: 700; color: var(--liki-primary); }
.zwc-row { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.zwc-sihua { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; margin-top: 2px; }
.zw-sihua-chip {
  display: inline-flex; align-items: center; gap: 3px; padding: 2px 8px; border: 1px solid var(--liki-line);
  border-radius: var(--radius-full); background: var(--liki-surface); font-size: var(--font-size-caption); color: var(--liki-text-2);
}
.zw-section { margin-top: 14px; }
.zw-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.zw-chip { padding: 3px 10px; border: 1px solid var(--liki-line); border-radius: var(--radius-full); background: var(--liki-surface); font-size: var(--font-size-caption); color: var(--liki-text-2); }
.zw-sanfang { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }
.zw-sf { border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: 8px 10px; background: var(--liki-surface); min-width: 0; }
.zw-sf-name { font-size: var(--font-size-caption); font-weight: 600; color: var(--liki-text-3); }
.zw-sf-body { font-size: var(--font-size-caption); color: var(--liki-ink); margin-top: 2px; }

.ln-pick { margin-bottom: 8px; }
.ln-sihua { justify-content: flex-start; }
.ln-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: repeat(4, minmax(60px, auto));
  gap: 1px;
  background: var(--liki-line);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-small);
  overflow: hidden;
  margin-top: 8px;
}
.ln-cell { background: var(--liki-surface); padding: 4px 5px; min-width: 0; }
.ln-cell.ming { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.ln-pname { font-size: var(--font-size-micro); font-weight: 600; color: var(--liki-text-2); white-space: nowrap; }
.ln-cell.ming .ln-pname { color: var(--liki-primary); }
.ln-stars { font-size: var(--font-size-micro); color: var(--liki-primary); line-height: 1.35; margin-top: 2px; }
.ln-center { grid-row: 2 / 4; grid-column: 2 / 4; background: var(--liki-surface-cream); display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 6px; text-align: center; }
.lnc-title { font-size: var(--font-size-body-lg); font-weight: 700; color: var(--liki-primary); }
.lnc-row { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: 2px; }

@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
  .zw-grid { grid-template-rows: repeat(4, minmax(88px, auto)); }
  .zw-palace { padding: 4px 3px; }
  .zw-pname { font-size: var(--font-size-micro); }
  .zw-center { padding: 6px 6px; gap: 4px; }
  .zwc-row { font-size: var(--font-size-micro); }
  .zw-sanfang { grid-template-columns: 1fr; }
  .ln-grid { grid-template-rows: repeat(4, minmax(52px, auto)); }
  .ln-cell { padding: 3px; }
}
</style>
