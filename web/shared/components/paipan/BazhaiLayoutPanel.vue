<template>
  <div
    v-if="bazhai || layout"
    class="bazhai-layout-panel"
  >
    <div
      v-if="bazhai"
      class="paipan-grid"
    >
      <template
        v-for="s in gridSlots"
        :key="s.slot"
      >
        <div
          v-if="s.center"
          class="bz-cell center"
        >
          <span class="bz-center">{{ $t('services.fengshui.centerPalace') }}</span>
        </div>
        <div
          v-else-if="s.cell"
          :class="['bz-cell', { good: s.cell.good }]"
        >
          <span class="bz-dir">{{ s.cell.direction }}</span>
          <span class="bz-gua font-serif">{{ s.cell.gua }}</span>
          <span class="bz-youxing">{{ s.cell.youxing }}</span>
          <span
            v-if="s.cell.badges.length"
            class="bz-badges"
          >
            <span
              v-for="badge in s.cell.badges"
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
    <div
      v-if="layout"
      class="bz-verdict"
    >
      <div class="bz-gua-line">
        <span class="bz-gua-name font-serif">{{ layout.ming_gua }}</span>
        <span class="bz-group">{{ layout.group }}</span>
      </div>
      <div class="mini-table-wrap">
        <table class="mini-table bz-table">
          <thead>
            <tr>
              <th scope="col" />
              <th scope="col">
                {{ $t('services.fengshui.directionLabel') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.guaLabel') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.youxing') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.rating') }}
              </th>
              <th scope="col">
                {{ $t('services.fengshui.match') }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in verdictRows"
              :key="row.key"
            >
              <th scope="row">
                {{ row.label }}
              </th>
              <td>{{ row.info.direction }}</td>
              <td>{{ row.info.gua_name }}</td>
              <td>{{ row.info.youxing }}</td>
              <td :class="row.good ? 'bz-good' : 'bz-bad'">
                {{ row.rating }}
              </td>
              <td :class="row.good ? 'bz-good' : 'bz-bad'">
                {{ row.match }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="bz-note">
        {{ $t('services.fengshui.layoutNote') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
// 八宅布局判断（bazhai_layout）——标准八宅方位图（九宫：方位/卦/游年星，四吉四凶着色）
// + 《阳宅三要》门/主/灶三要判断表。App 风水页与营销站 fengshui 岛共用。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BazhaiChart, BazhaiDoorStove, BazhaiLayout } from '../../api/engine/fengshui'
import { BAZHAI_AUS } from '../../utils/bazhai'

const props = defineProps<{
  layout: BazhaiLayout | null
  /** 命卦盘（提供 ba_zhai_dirs 以绘制方位图；可选）。 */
  bazhai?: BazhaiChart | null
}>()

const { t } = useI18n()

// 方位 → 洛书九宫槽位（上南下北：东南 南 西南 / 东 中宫 西 / 东北 北 西北）。
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

interface GridCell {
  slot: number
  direction: string
  gua: string
  youxing: string
  good: boolean
  badges: string[]
}

const gridCells = computed<GridCell[]>(() => {
  const b = props.bazhai
  if (!b) return []
  const cells: GridCell[] = []
  for (const [key, values] of Object.entries(b.ba_zhai_dirs)) {
    const direction = (values ?? [''])[0] ?? ''
    const slot = DIR_SLOT[direction]
    if (slot === undefined) continue
    cells.push({
      slot,
      direction,
      gua: DIR_GUA[direction] ?? '',
      youxing: t(`profiles.chart.direction.${key}`),
      good: BAZHAI_AUS.includes(key),
      badges: [],
    })
  }
  const layout = props.layout
  if (layout) {
    const addBadge = (info: BazhaiDoorStove, label: string) => {
      const cell = cells.find(c => c.slot === DIR_SLOT[info.direction])
      if (cell) cell.badges.push(label)
    }
    addBadge(layout.door, t('services.fengshui.door'))
    addBadge(layout.master, t('services.fengshui.master'))
    addBadge(layout.stove, t('services.fengshui.stove'))
  }
  return cells
})

function cellBySlot(slot: number): GridCell | undefined {
  return gridCells.value.find(c => c.slot === slot)
}

// 九宫槽位（0-8，行优先；4 为中宫）：上南下北。
const gridSlots = computed(() =>
  Array.from({ length: 9 }, (_, slot) => ({
    slot,
    center: slot === 4,
    cell: slot === 4 ? undefined : cellBySlot(slot),
  })),
)

const RATING_KEYS: Record<string, string> = {
  大吉: 'services.fengshui.ratingValue.veryAuspicious',
  吉: 'services.fengshui.ratingValue.auspicious',
  平: 'services.fengshui.ratingValue.neutral',
  凶: 'services.fengshui.ratingValue.inauspicious',
  大凶: 'services.fengshui.ratingValue.veryInauspicious',
}

function labelOf(keys: Record<string, string>, value: string): string {
  return keys[value] ? t(keys[value]) : value
}

const verdictRows = computed(() => {
  const l = props.layout
  if (!l) return []
  return [
    { key: 'door', label: t('services.fengshui.door'), info: l.door },
    { key: 'master', label: t('services.fengshui.master'), info: l.master },
    { key: 'stove', label: t('services.fengshui.stove'), info: l.stove },
  ].map(row => ({
    ...row,
    rating: labelOf(RATING_KEYS, row.info.rating),
    match: row.info.match === '吉' ? t('services.fengshui.ratingValue.auspicious') : row.info.match === '凶' ? t('services.fengshui.ratingValue.inauspicious') : row.info.match,
    good: row.info.match === '吉',
  }))
})
</script>

<style scoped src="../../styles/paipan.css"></style>

<style scoped>
.bz-cell { background: var(--liki-surface); padding: 8px 6px; text-align: center; min-height: 64px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1px; }
.bz-cell.good { background: var(--color-success-surface); }
.bz-cell.center { background: var(--liki-paper); }
.bz-center { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.bz-cell.empty { background: var(--liki-paper); }
.bz-dir { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.bz-gua { font-size: var(--font-size-title-sm); font-weight: 600; }
.bz-cell.good .bz-gua { color: var(--color-success); }
.bz-youxing { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.bz-badges { display: flex; flex-wrap: wrap; justify-content: center; gap: 3px; margin-top: 2px; }
.bz-badge { font-size: var(--font-size-micro); line-height: 1.3; padding: 0 6px; border-radius: var(--radius-full); background: var(--liki-primary); color: var(--liki-on-primary); }

.bz-verdict { margin-top: var(--space-3); }
.bz-gua-line { display: flex; align-items: baseline; gap: var(--space-2); margin-bottom: var(--space-2); }
.bz-gua-name { font-size: var(--font-size-title-sm); font-weight: 600; color: var(--liki-ink); }
.bz-group { font-size: var(--font-size-label); color: var(--liki-text-2); }
.bz-good { color: var(--color-success); }
.bz-bad { color: var(--color-error); }
.bz-note { font-size: var(--font-size-caption); color: var(--liki-text-2); margin: var(--space-2) 0 0; }
</style>
