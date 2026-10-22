<template>
  <div
    v-if="chart"
    class="xuankong-panel"
  >
    <div class="xk-head">
      <span class="xk-yuninfo">{{ chart.yun.yuan }} · {{ chart.yun.yun_name }}（{{ chart.yun.start_year }}–{{ chart.yun.end_year }}）</span>
      <span class="xk-sitface">{{ $t('services.fengshui.zuoShan') }} {{ chart.zuo_shan_name }} · {{ $t('services.fengshui.xiangShan') }} {{ chart.xiang_shan_name }}</span>
    </div>
    <div class="xk-legend">
      <span>{{ $t('services.fengshui.shanStar') }}</span>
      <span>{{ $t('services.fengshui.xiangStar') }}</span>
      <span>{{ $t('services.fengshui.yunStar') }}</span>
    </div>
    <div class="paipan-grid">
      <div
        v-for="p in palaces"
        :key="p.gong_num"
        :class="['xk-cell', { sit: p.gong_num === sitGong, face: p.gong_num === faceGong }]"
      >
        <div class="xk-stars">
          <span
            :class="['xk-star', { timely: p.shan_xing.number === chart.yun.yun_number }]"
            :title="p.shan_xing.name"
          >{{ p.shan_xing.number }}</span>
          <span
            :class="['xk-star', { timely: p.xiang_xing.number === chart.yun.yun_number }]"
            :title="p.xiang_xing.name"
          >{{ p.xiang_xing.number }}</span>
        </div>
        <div
          class="xk-yun"
          :title="p.yun_xing.name"
        >
          {{ cnNumber(p.yun_xing.number) }}
        </div>
        <div class="xk-badges">
          <span
            v-if="p.gong_num === sitGong"
            class="xk-badge sit"
          >{{ $t('services.fengshui.zuo') }}</span>
          <span
            v-if="p.gong_num === faceGong"
            class="xk-badge face"
          >{{ $t('services.fengshui.xiang') }}</span>
        </div>
      </div>
    </div>
    <div class="xk-sub">
      <span class="xk-sub-title">{{ $t('services.fengshui.fourSituation') }}</span>
      <span class="xk-situation">{{ chart.four_situation.name }}</span>
    </div>
    <div class="xk-timely">
      <span
        v-for="item in timelyItems"
        :key="item.label"
        :class="['paipan-chip', 'xk-timely-item', { on: item.on }]"
      >{{ item.label }}<i class="xk-mark">{{ item.on ? '✓' : '—' }}</i></span>
    </div>
    <div class="xk-sub">
      <span class="xk-sub-title">{{ $t('services.fengshui.shouShanChuSha') }}</span>
      <span class="xk-situation">{{ $t('services.fengshui.zhengShen') }} {{ chart.shou_shan_chu_sha.zheng_shen }} · {{ $t('services.fengshui.lingShen') }} {{ chart.shou_shan_chu_sha.ling_shen }}</span>
      <span :class="['xk-ok', { on: chart.shou_shan_chu_sha.shou_shan }]">{{ $t('services.fengshui.shouShanOk') }}<i class="xk-mark">{{ chart.shou_shan_chu_sha.shou_shan ? '✓' : '✗' }}</i></span>
      <span :class="['xk-ok', { on: chart.shou_shan_chu_sha.chu_sha }]">{{ $t('services.fengshui.chuShaOk') }}<i class="xk-mark">{{ chart.shou_shan_chu_sha.chu_sha ? '✓' : '✗' }}</i></span>
    </div>
    <p
      v-if="chart.shou_shan_chu_sha.assessment"
      class="xk-assessment"
    >
      {{ chart.shou_shan_chu_sha.assessment }}
    </p>
    <div
      v-if="fuYinLayers.length || fanYinLayers.length"
      class="xk-sub"
    >
      <template v-if="fuYinLayers.length">
        <span class="xk-sub-title">{{ $t('services.fengshui.fuYin') }}</span>
        <span
          v-for="layer in fuYinLayers"
          :key="layer"
          class="paipan-chip"
        >{{ layer }}</span>
      </template>
      <template v-if="fanYinLayers.length">
        <span class="xk-sub-title">{{ $t('services.fengshui.fanYin') }}</span>
        <span
          v-for="layer in fanYinLayers"
          :key="layer"
          class="paipan-chip"
        >{{ layer }}</span>
      </template>
    </div>
    <div
      v-if="xingJiaHui.length"
      class="xk-combos"
    >
      <div class="xk-sub-title">
        {{ $t('services.fengshui.xingJiaHui') }}
      </div>
      <div
        v-for="(x, i) in xingJiaHui"
        :key="i"
        class="xk-combo"
      >
        <span class="xk-combo-name">{{ x.name }}</span>
        <span class="xk-combo-stars">山{{ x.shan_num }} · 向{{ x.xiang_num }}</span>
        <span class="xk-combo-meaning">{{ x.meaning }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 玄空飞星宅盘（xuankong_chart）——标准九宫排布：山星左上、向星右上（阿拉伯数字），
// 运星居中下（中文数字）；坐/向角标；当令星主色。补充四大格局/收山出煞/伏吟反吟/星加会。
// App 风水页与营销站 fengshui 岛共用；文案取 services.fengshui.*（共享拷贝见 chart-messages.<locale>.json）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { XuankongChart } from '../../api/engine'
import { LUOSHU_ORDER } from '../../utils/bazhai'
import { cnNumber, mountainGong } from '../../utils/xuankong'

const props = defineProps<{ chart: XuankongChart | null }>()

const { t } = useI18n()

const palaces = computed(() => {
  const c = props.chart
  if (!c) return []
  return LUOSHU_ORDER.map(n => c.gong_wei.find(p => p.gong_num === n)).filter(Boolean) as XuankongChart['gong_wei']
})

const sitGong = computed(() => (props.chart ? mountainGong(props.chart.zuo_shan) : 0))
const faceGong = computed(() => (props.chart ? mountainGong(props.chart.xiang_shan) : 0))

const timelyItems = computed(() => {
  const f = props.chart?.four_situation
  if (!f) return []
  return [
    { label: t('services.fengshui.sitMountainTimely'), on: f.sit_mountain_timely },
    { label: t('services.fengshui.faceFacingTimely'), on: f.face_facing_timely },
    { label: t('services.fengshui.sitFacingTimely'), on: f.sit_facing_timely },
    { label: t('services.fengshui.faceMountainTimely'), on: f.face_mountain_timely },
  ]
})

const fuYinLayers = computed(() => props.chart?.fu_yin_layers ?? [])
const fanYinLayers = computed(() => props.chart?.fan_yin_layers ?? [])

// 星加会仅显示命名组合（引擎对未列入固定通则的组合返回 fallback 名「星曜加会」）。
const xingJiaHui = computed(() =>
  (props.chart?.xing_jia_hui ?? []).filter(x => x.name && x.name !== '星曜加会'),
)
</script>

<style scoped src="../../styles/paipan.css"></style>

<style scoped>
.xk-head { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-3); align-items: baseline; font-size: var(--font-size-label); color: var(--liki-text-2); margin-bottom: var(--space-2); }
.xk-yuninfo { font-weight: 600; color: var(--liki-ink); }
.xk-legend { display: flex; gap: var(--space-3); font-size: var(--font-size-micro); color: var(--liki-text-3); margin-bottom: var(--space-2); }
.xk-cell { background: var(--liki-surface); padding: var(--space-2); }
.xk-cell.sit, .xk-cell.face { background: var(--liki-surface-cream); }
.xk-stars { display: flex; justify-content: space-between; align-items: baseline; }
.xk-star { font-size: var(--font-size-title-sm); font-weight: 700; font-variant-numeric: tabular-nums; }
.xk-star.timely { color: var(--liki-primary); }
.xk-yun { font-size: var(--font-size-caption); color: var(--liki-text-3); text-align: center; margin-top: 2px; }
.xk-badges { display: flex; justify-content: center; gap: 3px; margin-top: 3px; min-height: 15px; }
.xk-badge { font-size: var(--font-size-micro); line-height: 1.3; padding: 0 6px; border-radius: var(--radius-full); }
.xk-badge.sit { background: var(--liki-primary); color: var(--liki-on-primary); }
.xk-badge.face { border: 1px solid var(--liki-primary); color: var(--liki-primary); }

.xk-sub { display: flex; flex-wrap: wrap; align-items: baseline; gap: var(--space-2) var(--space-3); margin-top: var(--space-3); font-size: var(--font-size-label); }
.xk-sub-title { color: var(--liki-text-3); }
.xk-situation { font-weight: 600; color: var(--liki-ink); }
.xk-timely { display: flex; flex-wrap: wrap; gap: var(--space-2); margin-top: var(--space-2); }
.xk-timely-item.on { color: var(--liki-primary); border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.xk-mark { font-style: normal; margin-left: var(--space-1); }
.xk-ok { font-size: var(--font-size-caption); color: var(--liki-text-3); }
.xk-ok.on { color: var(--color-success); }
.xk-assessment { font-size: var(--font-size-caption); color: var(--liki-text-2); margin: var(--space-1) 0 0; }
.xk-combos { margin-top: var(--space-3); }
.xk-combo { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-3); font-size: var(--font-size-caption); margin-top: var(--space-1); }
.xk-combo-name { color: var(--liki-ink); font-weight: 600; }
.xk-combo-stars { color: var(--liki-text-3); }
.xk-combo-meaning { color: var(--liki-text-2); }
</style>
