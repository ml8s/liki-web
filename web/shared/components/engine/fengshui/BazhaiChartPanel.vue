<template>
  <div
    class="bazhai-chart-panel"
    :data-density="density ?? 'full'"
  >
    <div class="mingua-row">
      <div class="mingua font-serif">
        {{ result.ming_gua.gua.name }}
      </div>
      <div>
        <div class="mg-name">
          {{ result.ming_gua.group }}
        </div>
        <div
          v-if="result.ming_gua.gua.wuxing"
          class="mg-wuxing"
        >
          {{ $t('services.fengshui.element') }}{{ result.ming_gua.gua.wuxing }}
        </div>
      </div>
    </div>

    <div
      class="dirs"
      role="grid"
      :aria-label="$t('services.fengshui.directionLabel')"
    >
      <div
        v-for="(row, rowIndex) in directionRows"
        :key="rowIndex"
        class="grid-row"
        role="row"
      >
        <div
          v-for="cell in row"
          :key="cell.key"
          :class="['dir', { good: cell.good, 'is-focused': cell.key === focused }]"
          role="gridcell"
          tabindex="0"
          :data-focus-key="cell.key"
          :aria-label="directionAriaLabel(cell)"
          :aria-current="cell.key === focused ? 'true' : undefined"
        >
          <div class="dg font-serif">
            {{ cell.gua }}
          </div>
          <div class="dname">
            {{ $t(`profiles.chart.direction.${cell.key}`) }}
          </div>
        </div>
      </div>
    </div>

    <div
      class="chart-grid bz-grid"
      role="grid"
      :aria-label="$t('services.fengshui.directionLabel')"
    >
      <template
        v-for="slot in layoutGridSlots"
        :key="slot.slot"
      >
        <div
          v-if="slot.center"
          class="bz-cell center"
        >
          <span class="bz-center">{{ $t('services.fengshui.centerPalace') }}</span>
        </div>
        <div
          v-else-if="slot.cell"
          :class="['bz-cell', { good: slot.cell.good }]"
        >
          <span class="bz-dir">{{ slot.cell.direction }}</span>
          <span class="bz-gua font-serif">{{ slot.cell.gua }}</span>
          <span class="bz-youxing">{{ slot.cell.youxing }}</span>
          <span
            v-if="slot.cell.badges.length"
            class="bz-badges"
          >
            <span
              v-for="badge in slot.cell.badges"
              :key="badge"
              class="bz-badge"
            >{{ badge }}</span>
          </span>
        </div>
        <div
          v-else
          class="bz-cell empty"
        />
      </template>
    </div>

    <template v-if="liuNian">
      <div class="sec-title">
        {{ $t('profiles.chart.annualStars', { year: liuNian.year }) }}
      </div>
      <div class="ln-row">
        {{ $t('profiles.chart.centerStar') }}<strong>{{ liuNian.ru_zhong }}</strong>
      </div>
      <div
        class="chart-grid"
        role="grid"
        :aria-label="$t('profiles.chart.annualStars', { year: liuNian.year })"
      >
        <div
          v-for="(row, rowIndex) in annualStarRows"
          :key="rowIndex"
          class="grid-row"
          role="row"
        >
          <div
            v-for="cell in row"
            :key="cell.gong_num"
            :class="['fly-cell', { center: cell.gong_num === 5, ru_zhong: cell.ru_zhong, 'is-focused': annualFocusKey(cell) === focused }]"
            role="gridcell"
            tabindex="0"
            :data-focus-key="annualFocusKey(cell)"
            :aria-label="annualStarAriaLabel(cell)"
            :aria-current="annualFocusKey(cell) === focused ? 'true' : undefined"
          >
            <div class="fly-name">
              {{ cell.xing_name }}
            </div>
            <div
              v-if="cell.wuxing"
              class="fly-wuxing"
            >
              {{ cell.wuxing }}
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BazhaiChartPanelProps } from '../contract'
import { BAZHAI_AUS, LUOSHU_ORDER } from '../../../utils/bazhai'

const props = defineProps<BazhaiChartPanelProps>()

const { t } = useI18n()

interface DirectionCell {
  key: string
  gua: string
  good: boolean
}

interface LayoutGridCell {
  slot: number
  direction: string
  gua: string
  youxing: string
  good: boolean
  badges: string[]
}

const DIR_SLOT: Record<string, number> = {
  东南: 0, 正南: 1, 南: 1, 西南: 2,
  正东: 3, 东: 3, 正西: 5, 西: 5,
  东北: 6, 正北: 7, 北: 7, 西北: 8,
}
const DIR_GUA: Record<string, string> = {
  东南: '巽', 正南: '离', 南: '离', 西南: '坤',
  正东: '震', 东: '震', 正西: '兑', 西: '兑',
  东北: '艮', 正北: '坎', 北: '坎', 西北: '乾',
}

interface AnnualStarCell {
  gong_num: number
  xing_name: string
  wuxing?: string
  ru_zhong?: boolean
}

const directionCells = computed<DirectionCell[]>(() =>
  Object.entries(props.result.ba_zhai_dirs).map(([key, direction]) => ({
    key,
    gua: direction ?? '',
    good: BAZHAI_AUS.includes(key),
  })),
)

const directionRows = computed(() => rowsOf(directionCells.value, 4))

const layoutGridCells = computed<LayoutGridCell[]>(() => {
  const cells: LayoutGridCell[] = []
  for (const [key, direction] of Object.entries(props.result.ba_zhai_dirs)) {
    const slot = DIR_SLOT[direction ?? '']
    if (slot === undefined) continue
    cells.push({
      slot,
      direction: direction ?? '',
      gua: DIR_GUA[direction ?? ''] ?? '',
      youxing: t(`profiles.chart.direction.${key}`),
      good: BAZHAI_AUS.includes(key),
      badges: [],
    })
  }
  const layout = props.layout
  if (!layout) return cells
  const addBadge = (info: NonNullable<BazhaiChartPanelProps['layout']>['door'], label: string) => {
    cells.find(cell => cell.slot === DIR_SLOT[info.direction])?.badges.push(label)
  }
  addBadge(layout.door, t('services.fengshui.door'))
  addBadge(layout.master, t('services.fengshui.master'))
  addBadge(layout.stove, t('services.fengshui.stove'))
  return cells
})

const layoutGridSlots = computed(() => Array.from({ length: 9 }, (_, slot) => {
  const cell = layoutGridCells.value.find(candidate => candidate.slot === slot)
  return { slot, cell, center: slot === 4 }
}))

const liuNian = computed(() => props.result.liu_nian_xing)

const annualStarCells = computed<AnnualStarCell[]>(() => {
  const gong = liuNian.value?.gong_wei
  if (!gong || gong.length !== 9) return []
  return LUOSHU_ORDER.map(n => gong.find(item => item.gong_num === n) ?? {
    gong_num: n,
    xing_name: '—',
  })
})

const annualStarRows = computed(() => rowsOf(annualStarCells.value, 3))

function rowsOf<T>(items: T[], columns: number): T[][] {
  return Array.from({ length: Math.ceil(items.length / columns) }, (_, index) =>
    items.slice(index * columns, (index + 1) * columns))
}

function annualFocusKey(cell: AnnualStarCell): string {
  return `liu_nian_${cell.gong_num}`
}

function directionAriaLabel(cell: DirectionCell): string {
  return [
    cell.gua,
    t(`profiles.chart.direction.${cell.key}`),
    cell.good ? t('profiles.chart.auspicious') : t('profiles.chart.inauspicious'),
  ].filter(Boolean).join(' · ')
}

function annualStarAriaLabel(cell: AnnualStarCell): string {
  return [
    cell.xing_name,
    cell.wuxing,
    cell.ru_zhong ? t('profiles.chart.centerStar') : '',
  ].filter(Boolean).join(' · ')
}
</script>

<style scoped>
.bazhai-chart-panel {
  min-width: 0;
  overflow-wrap: anywhere;
}

.mingua-row { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-4); margin-bottom: var(--space-4); }
.mingua { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 50%; border: 2px solid var(--liki-primary); font-size: var(--font-size-title-lg); color: var(--liki-primary); background: var(--liki-primary-tint); }
.mg-name { font-size: var(--font-size-body-lg); font-weight: 600; overflow-wrap: anywhere; }
.mg-wuxing { font-size: var(--font-size-label); color: var(--liki-text-2); margin-top: 2px; }

.dirs { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-2); min-width: 0; }
.grid-row { display: contents; }
.dir { min-width: 0; border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: var(--space-3); text-align: center; overflow-wrap: anywhere; }
.dir .dg { font-size: var(--font-size-title-sm); color: var(--liki-primary); }
.dir.good { border-color: var(--color-success-border); background: var(--color-success-surface); }
.dir.good .dg { color: var(--color-success); }
.dir.is-focused { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.dir .dname { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: 3px; }
.bz-grid { margin-top: var(--space-4); grid-template-columns: repeat(3, minmax(0, 1fr)); }
.bz-cell { min-width: 0; min-height: 68px; padding: var(--space-2); border: 1px solid var(--liki-line); border-radius: var(--radius-small); text-align: center; overflow-wrap: anywhere; }
.bz-cell.good { border-color: var(--color-success-border); background: var(--color-success-surface); }
.bz-cell.center { display: grid; place-items: center; color: var(--liki-text-2); }
.bz-dir { display: block; font-size: var(--font-size-caption); color: var(--liki-text-2); }
.bz-gua { display: block; font-size: var(--font-size-title-sm); color: var(--liki-primary); }
.bz-youxing { display: block; font-size: var(--font-size-label); }
.bz-badges { display: flex; flex-wrap: wrap; justify-content: center; gap: var(--space-1); }
.bz-badge { padding: 1px 6px; border: 1px solid var(--liki-primary); border-radius: var(--radius-full); font-size: var(--font-size-micro); color: var(--liki-primary); }
.fly-cell { min-width: 0; background: var(--liki-surface); padding: var(--space-3) var(--space-1); text-align: center; overflow-wrap: anywhere; }
.fly-cell.center { background: var(--liki-primary-tint); }
.fly-cell.ru_zhong { border: 2px solid var(--liki-primary); background: var(--liki-primary-tint); }
.fly-cell.is-focused { border-color: var(--liki-primary-border); }
.fly-name { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.fly-wuxing { font-size: var(--font-size-micro); color: var(--liki-text-3); margin-top: 2px; }

[role='gridcell']:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: -2px;
}

.bazhai-chart-panel[data-density='compact'] .mingua { width: 44px; height: 44px; font-size: var(--font-size-title-sm); }
.bazhai-chart-panel[data-density='compact'] .dir { padding: var(--space-2); }
.bazhai-chart-panel[data-density='compact'] .fly-cell { padding: var(--space-2) var(--space-1); }

@media (max-width: 640px) {
  .dirs { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
