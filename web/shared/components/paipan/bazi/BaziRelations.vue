<template>
  <template v-if="relationRows.length">
    <div class="sec-title">{{ $t('profiles.chart.relations') }}</div>
    <div class="rel-table">
      <div
        v-for="row in relationRows"
        :key="row.label"
        class="rel-row"
      >
        <span class="rel-label">{{ row.label }}</span>
        <span class="rel-chips">
          <span
            v-for="(c, i) in row.items"
            :key="i"
            class="paipan-chip"
          >
            <span class="rel-main font-serif">{{ c.text }}</span>
            <span
              v-if="c.el"
              class="rel-el"
            >→{{ c.el }}</span>
            <span
              v-if="c.pos"
              class="rel-pos"
            >{{ c.pos }}</span>
            <span
              v-if="c.mark"
              class="rel-mark"
            >{{ c.mark }}</span>
          </span>
        </span>
      </div>
    </div>
  </template>
  <div
    v-if="tenGodRows.length || elementRows.length"
    class="analysis-grid"
  >
    <section
      v-if="tenGodRows.length"
      class="analysis-sec"
    >
      <div class="sec-title">{{ $t('profiles.chart.tenGods') }}</div>
      <div class="mini-table-wrap">
        <table class="mini-table">
          <thead>
            <tr>
              <th scope="col" />
              <th scope="col">{{ $t('profiles.chart.marks.transparent') }}</th>
              <th scope="col">{{ $t('profiles.chart.marks.hidden') }}</th>
              <th scope="col">{{ $t('profiles.chart.marks.rooted') }}</th>
              <th scope="col">{{ $t('profiles.chart.marks.timely') }}</th>
              <th
                scope="col"
                class="num"
              >{{ $t('profiles.chart.count') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="s in tenGodRows"
              :key="s.shi_shen"
              :class="`tg-${s.strength}`"
            >
              <th scope="row">
                <span class="tg-name">{{ s.shi_shen }}</span><span class="tg-wx">{{ s.wuxing }}</span>
              </th>
              <td><i :class="['dot', { on: s.transparent }]" /></td>
              <td><i :class="['dot', { on: s.hidden }]" /></td>
              <td><i :class="['dot', { on: s.rooted }]" /></td>
              <td><i :class="['dot', { on: s.timely }]" /></td>
              <td class="num">{{ s.count }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <section
      v-if="elementRows.length"
      class="analysis-sec"
    >
      <div class="sec-title">{{ $t('profiles.chart.elementStates') }}</div>
      <div class="mini-table-wrap">
        <table class="mini-table">
          <thead>
            <tr>
              <th scope="col" />
              <th scope="col">{{ $t('profiles.chart.season') }}</th>
              <th scope="col">{{ $t('profiles.chart.strength') }}</th>
              <th scope="col">{{ $t('profiles.chart.marks.transparent') }}</th>
              <th scope="col">{{ $t('profiles.chart.marks.rooted') }}</th>
              <th
                scope="col"
                class="num"
              >{{ $t('profiles.chart.count') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="e in elementRows"
              :key="e.wuxing"
              :class="`el-${e.strength}`"
            >
              <th scope="row">
                <span class="tg-name">{{ e.wuxing }}</span>
              </th>
              <td>{{ mapped(SEASON_KEYS, e.season) }}</td>
              <td>{{ mapped(STRENGTH_KEYS, e.strength) }}</td>
              <td><i :class="['dot', { on: e.transparent }]" /></td>
              <td><i :class="['dot', { on: e.rooted }]" /></td>
              <td class="num">{{ wuxingCounts[e.wuxing] ?? '' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
  <div
    v-if="luRoots.length"
    class="shensha"
  >
    <span class="sy-label">{{ $t('profiles.chart.luRoots') }}</span>
    <span
      v-for="r in luRoots"
      :key="`${r.gan}${r.zhi}`"
      class="paipan-chip"
    >
      <span class="font-serif">{{ r.gan }}禄{{ r.zhi }}</span><span class="lu-shen">{{ r.shi_shen }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziFullChartData, BaziJuRelation } from '../../../api/engine/bazi'

const props = defineProps<{ full?: BaziFullChartData | null }>()
const { t } = useI18n()

const PILLAR_KEYS = ['year', 'month', 'day', 'hour'] as const
const ENGINE_PILLAR_INDEX: Record<string, number> = { nian: 0, yue: 1, ri: 2, shi: 3 }

function pillarShort(index: number): string {
  const key = PILLAR_KEYS[index]
  return key ? t(`profiles.chart.pillarShort.${key}`) : ''
}
function pillarIndex(key: string): number {
  return ENGINE_PILLAR_INDEX[key] ?? -1
}
function posText(...indices: number[]): string {
  return indices.map(pillarShort).filter(Boolean).join('·')
}
const SEASON_KEYS: Record<string, string> = {
  旺: 'profiles.chart.seasonValue.prosperous',
  相: 'profiles.chart.seasonValue.strong',
  休: 'profiles.chart.seasonValue.resting',
  囚: 'profiles.chart.seasonValue.trapped',
  死: 'profiles.chart.seasonValue.dead',
}
const STRENGTH_KEYS: Record<string, string> = {
  strong: 'profiles.chart.strengthValue.strong',
  neutral: 'profiles.chart.strengthValue.neutral',
  weak: 'profiles.chart.strengthValue.weak',
}
function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}

interface RelationChip {
  text: string
  el?: string
  pos?: string
  mark?: string
}

const relationRows = computed(() => {
  const f = props.full
  if (!f) return []
  const rows: Array<{ label: string; items: RelationChip[] }> = []
  const add = (label: string, items: RelationChip[]) => {
    const list = items.filter(i => i.text)
    if (list.length) rows.push({ label, items: list })
  }
  const pair = (r: { zhi_a: string; zhi_b: string; pillar_a: number; pillar_b: number }): RelationChip => ({
    text: `${r.zhi_a}${r.zhi_b}`,
    pos: posText(r.pillar_a, r.pillar_b),
  })
  const ju = (r: BaziJuRelation): RelationChip => ({
    text: r.name ?? '',
    pos: (r.pillars ?? []).map(p => pillarShort(pillarIndex(p))).filter(Boolean).join('·'),
  })

  add(t('profiles.chart.rel.ganHe'), (f.gan_he ?? []).map(r => ({
    text: `${r.gan_a}${r.gan_b}`,
    el: r.he_element,
    pos: posText(r.pillar_a, r.pillar_b),
    mark: r.contested ? t('profiles.chart.rel.contested') : undefined,
  })))
  add(t('profiles.chart.rel.ganChong'), (f.gan_chong ?? []).map(r => ({
    text: `${r.gan_a}${r.gan_b}`,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  add(t('profiles.chart.rel.zhiLiuHe'), (f.zhi_liu_he ?? []).map(r => ({
    text: `${r.zhi_a}${r.zhi_b}`,
    el: r.wuxing,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  add(t('profiles.chart.rel.sanHe'), (f.san_he ?? []).map(ju))
  add(t('profiles.chart.rel.sanHePartial'), (f.san_he_partial ?? []).map(ju))
  add(t('profiles.chart.rel.sanHui'), (f.san_hui ?? []).map(ju))
  add(t('profiles.chart.rel.liuChong'), (f.liu_chong ?? []).map(pair))
  add(t('profiles.chart.rel.liuHai'), (f.liu_hai ?? []).map(pair))
  add(t('profiles.chart.rel.liuXing'), (f.liu_xing ?? []).map(pair))
  add(t('profiles.chart.rel.liuPo'), (f.liu_po ?? []).map(pair))
  add(t('profiles.chart.rel.anHe'), (f.an_he ?? []).map(pair))
  add(t('profiles.chart.rel.gongJia'), (f.gong_jia ?? []).map(r => ({
    text: `${r.type}${r.zhi}`,
    el: r.wuxing,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  return rows
})

const tenGodRows = computed(() => props.full?.ten_god_states ?? [])
const elementRows = computed(() => props.full?.element_states ?? [])
const wuxingCounts = computed(() => props.full?.fu_yi?.wuxing_count ?? {})

const luRoots = computed(() => {
  const seen = new Set<string>()
  return (props.full?.lu_roots ?? []).filter(r => {
    const key = `${r.gan}${r.zhi}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
})
</script>

<style scoped src="../../../styles/paipan.css"></style>

<style scoped>
.rel-table { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.rel-row { display: grid; grid-template-columns: minmax(5.5em, max-content) 1fr; gap: var(--space-2); padding: var(--space-2) var(--space-3); font-size: var(--font-size-label); align-items: baseline; }
.rel-row + .rel-row { border-top: 1px solid var(--liki-line); }
.rel-label { color: var(--liki-text-3); }
.rel-chips { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.rel-main { color: var(--liki-ink); font-weight: 600; }
.rel-el { color: var(--liki-primary); }
.rel-pos { color: var(--liki-text-3); font-size: var(--font-size-micro); }
.rel-mark { color: var(--color-error); font-size: var(--font-size-micro); }
.analysis-grid { display: grid; gap: var(--space-2); }
.analysis-sec { min-width: 0; }
.mini-table .num { font-variant-numeric: tabular-nums; color: var(--liki-text-2); }
.tg-wx { margin-left: var(--space-1); font-size: var(--font-size-micro); color: var(--liki-text-3); font-weight: 400; }
.dot { display: inline-block; width: 7px; height: 7px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); }
.dot.on { background: var(--liki-primary); border-color: var(--liki-primary); }
.tg-strong .tg-name, .el-strong .tg-name { color: var(--liki-primary); }
.tg-weak .tg-name, .el-weak .tg-name { color: var(--liki-text-3); }
.shensha { display: flex; flex-wrap: wrap; gap: var(--space-2); align-items: center; margin-top: var(--space-3); font-size: var(--font-size-label); }
.sy-label { color: var(--liki-text-3); flex: 0 0 auto; min-width: 2em; }
.lu-shen { margin-left: var(--space-1); color: var(--liki-text-3); font-size: var(--font-size-micro); }
@media (min-width: 640px) {
  .analysis-grid { grid-template-columns: 1fr 1fr; gap: var(--space-4); }
}
</style>
