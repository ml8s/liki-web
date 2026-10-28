<template>
  <div
    class="xuankong-panel"
    :data-density="density ?? 'full'"
  >
    <div class="xk-head">
      <span class="xk-yuninfo">{{ yunInfo }}</span>
      <span class="xk-sitface">{{ $t('services.fengshui.zuoShan') }} {{ result.zuo_shan_name }} · {{ $t('services.fengshui.xiangShan') }} {{ result.xiang_shan_name }}</span>
    </div>
    <div class="xk-legend">
      <span>{{ $t('services.fengshui.shanStar') }}</span>
      <span>{{ $t('services.fengshui.xiangStar') }}</span>
      <span>{{ $t('services.fengshui.yunStar') }}</span>
    </div>
    <div
      class="chart-grid"
      role="grid"
      :aria-label="$t('services.fengshui.xuankongTitle')"
    >
      <div
        v-for="(row, rowIndex) in palaceRows"
        :key="rowIndex"
        class="grid-row"
        role="row"
      >
        <div
          v-for="palace in row"
          :key="palace.gong_num"
          :class="['xk-cell', { sit: palace.gong_num === sitGong, face: palace.gong_num === faceGong, 'is-focused': palaceFocusKey(palace) === focused }]"
          role="gridcell"
          tabindex="0"
          :data-focus-key="palaceFocusKey(palace)"
          :aria-label="palaceAriaLabel(palace)"
          :aria-current="palaceFocusKey(palace) === focused ? 'true' : undefined"
        >
          <div class="xk-stars">
            <span
              :class="['xk-star', { timely: palace.shan_xing.number === result.yun.yun_number }]"
              :title="palace.shan_xing.name"
            >{{ palace.shan_xing.number }}</span>
            <span
              :class="['xk-star', { timely: palace.xiang_xing.number === result.yun.yun_number }]"
              :title="palace.xiang_xing.name"
            >{{ palace.xiang_xing.number }}</span>
          </div>
          <div
            class="xk-yun"
            :title="palace.yun_xing.name"
          >
            {{ cnNumber(palace.yun_xing.number) }}
          </div>
          <div class="xk-badges">
            <span
              v-if="palace.gong_num === sitGong"
              class="xk-badge sit"
            >{{ $t('services.fengshui.zuo') }}</span>
            <span
              v-if="palace.gong_num === faceGong"
              class="xk-badge face"
            >{{ $t('services.fengshui.xiang') }}</span>
          </div>
        </div>
      </div>
    </div>
    <div class="xk-sub">
      <span class="xk-sub-title">{{ $t('services.fengshui.fourSituation') }}</span>
      <span class="xk-situation">{{ result.four_situation.name }}</span>
    </div>
    <div class="xk-timely">
      <span
        v-for="item in timelyItems"
        :key="item.label"
        :class="['chart-chip', 'xk-timely-item', { on: item.on }]"
      >{{ item.label }}<i class="xk-mark">{{ item.on ? '✓' : '—' }}</i></span>
    </div>
    <div class="xk-sub">
      <span class="xk-sub-title">{{ $t('services.fengshui.shouShanChuSha') }}</span>
      <span class="xk-situation">{{ $t('services.fengshui.zhengShen') }} {{ result.shou_shan_chu_sha.zheng_shen }} · {{ $t('services.fengshui.lingShen') }} {{ result.shou_shan_chu_sha.ling_shen }}</span>
      <span :class="['xk-ok', { on: result.shou_shan_chu_sha.shou_shan }]">{{ $t('services.fengshui.shouShanOk') }}<i class="xk-mark">{{ result.shou_shan_chu_sha.shou_shan ? '✓' : '✗' }}</i></span>
      <span :class="['xk-ok', { on: result.shou_shan_chu_sha.chu_sha }]">{{ $t('services.fengshui.chuShaOk') }}<i class="xk-mark">{{ result.shou_shan_chu_sha.chu_sha ? '✓' : '✗' }}</i></span>
    </div>
    <p
      v-if="result.shou_shan_chu_sha.assessment"
      class="xk-assessment"
    >
      {{ result.shou_shan_chu_sha.assessment }}
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
          class="chart-chip"
        >{{ layer }}</span>
      </template>
      <template v-if="fanYinLayers.length">
        <span class="xk-sub-title">{{ $t('services.fengshui.fanYin') }}</span>
        <span
          v-for="layer in fanYinLayers"
          :key="layer"
          class="chart-chip"
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
        v-for="(combination, index) in xingJiaHui"
        :key="index"
        class="xk-combo"
      >
        <span class="xk-combo-name">{{ combination.name }}</span>
        <span class="xk-combo-stars">山{{ combination.shan_num }} · 向{{ combination.xiang_num }}</span>
        <span class="xk-combo-meaning">{{ combination.meaning }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { XuankongChartPanelProps } from '../contract'
import { LUOSHU_ORDER } from '../../../utils/bazhai'
import { cnNumber, mountainGong } from '../../../utils/xuankong'

const props = defineProps<XuankongChartPanelProps>()

const { t } = useI18n()

type XuankongPalace = XuankongChartPanelProps['result']['gong_wei'][number]

const yunInfo = computed(() => {
  const yun = props.result.yun
  return `${[yun.yuan, yun.yun_name].filter(Boolean).join(' · ')}（${yun.start_year}–${yun.end_year}）`
})

const palaces = computed(() => {
  return LUOSHU_ORDER
    .map(number => props.result.gong_wei.find(palace => palace.gong_num === number))
    .filter((palace): palace is XuankongPalace => Boolean(palace))
})

const palaceRows = computed(() => rowsOf(palaces.value, 3))
const sitGong = computed(() => mountainGong(props.result.zuo_shan))
const faceGong = computed(() => mountainGong(props.result.xiang_shan))

const timelyItems = computed(() => {
  const situation = props.result.four_situation
  return [
    { label: t('services.fengshui.sitMountainTimely'), on: situation.sit_mountain_timely },
    { label: t('services.fengshui.faceFacingTimely'), on: situation.face_facing_timely },
    { label: t('services.fengshui.sitFacingTimely'), on: situation.sit_facing_timely },
    { label: t('services.fengshui.faceMountainTimely'), on: situation.face_mountain_timely },
  ]
})

const fuYinLayers = computed(() => props.result.fu_yin_layers ?? [])
const fanYinLayers = computed(() => props.result.fan_yin_layers ?? [])

// 星加会仅显示命名组合（引擎对未列入固定通则的组合返回 fallback 名「星曜加会」）。
const xingJiaHui = computed(() =>
  (props.result.xing_jia_hui ?? []).filter(item => item.name && item.name !== '星曜加会'),
)

function rowsOf<T>(items: T[], columns: number): T[][] {
  return Array.from({ length: Math.ceil(items.length / columns) }, (_, index) =>
    items.slice(index * columns, (index + 1) * columns))
}

function palaceFocusKey(palace: XuankongPalace): string {
  return `gong_${palace.gong_num}`
}

function palaceAriaLabel(palace: XuankongPalace): string {
  const labels = [
    `${t('services.fengshui.shanStar')} ${palace.shan_xing.number} ${palace.shan_xing.name}`,
    `${t('services.fengshui.xiangStar')} ${palace.xiang_xing.number} ${palace.xiang_xing.name}`,
    `${t('services.fengshui.yunStar')} ${palace.yun_xing.number} ${palace.yun_xing.name}`,
  ]
  if (palace.gong_num === sitGong.value) labels.push(t('services.fengshui.zuo'))
  if (palace.gong_num === faceGong.value) labels.push(t('services.fengshui.xiang'))
  return labels.join(' · ')
}
</script>

<style scoped>
.xuankong-panel {
  min-width: 0;
  overflow-wrap: anywhere;
}

.xk-head { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-3); align-items: baseline; font-size: var(--font-size-label); color: var(--liki-text-2); margin-bottom: var(--space-2); }
.xk-yuninfo { font-weight: 600; color: var(--liki-ink); }
.xk-legend { display: flex; flex-wrap: wrap; gap: var(--space-3); font-size: var(--font-size-micro); color: var(--liki-text-3); margin-bottom: var(--space-2); }
.grid-row { display: contents; }
.xk-cell { min-width: 0; background: var(--liki-surface); padding: var(--space-2); overflow-wrap: anywhere; }
.xk-cell.sit, .xk-cell.face { background: var(--liki-surface-cream); }
.xk-cell.is-focused { border: 1px solid var(--liki-primary-border); }
.xk-stars { display: flex; justify-content: space-between; align-items: baseline; gap: var(--space-1); }
.xk-star { font-size: var(--font-size-title-sm); font-weight: 700; font-variant-numeric: tabular-nums; }
.xk-star.timely { color: var(--liki-primary); }
.xk-yun { font-size: var(--font-size-caption); color: var(--liki-text-3); text-align: center; margin-top: 2px; }
.xk-badges { display: flex; flex-wrap: wrap; justify-content: center; gap: 3px; margin-top: 3px; min-height: 15px; }
.xk-badge { font-size: var(--font-size-micro); line-height: 1.3; padding: 0 6px; border-radius: var(--radius-full); }
.xk-badge.sit { background: var(--liki-primary); color: var(--liki-on-primary); }
.xk-badge.face { border: 1px solid var(--liki-primary); color: var(--liki-primary); }

.xk-sub { display: flex; flex-wrap: wrap; align-items: baseline; gap: var(--space-2) var(--space-3); margin-top: var(--space-3); font-size: var(--font-size-label); }
.xk-sub-title { color: var(--liki-text-3); }
.xk-situation { font-weight: 600; color: var(--liki-ink); overflow-wrap: anywhere; }
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

[role='gridcell']:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: -2px;
}

.xuankong-panel[data-density='compact'] .xk-cell { padding: var(--space-1); }
.xuankong-panel[data-density='compact'] .xk-sub { margin-top: var(--space-2); font-size: var(--font-size-caption); }

@media (max-width: 320px) {
  .xk-head,
  .xk-sub,
  .xk-combo { display: block; }
}
</style>
