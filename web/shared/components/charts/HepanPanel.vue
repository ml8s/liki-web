<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 合盘结果盘面（共享展示组件）：两盘四柱对照 + 日主/地支/纳音/神煞/十神互看。
// App HepanView 与营销站 hepan 岛共用同一实现；文案取自共享消息拷贝
// （profiles.chart.* / services.hepan.*）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziBondResult, BaziChartData } from '../../api/engine/bazi'

const props = defineProps<{
  chartA: BaziChartData | null
  chartB: BaziChartData | null
  bond: BaziBondResult | null
  /** A 方展示名（App 传档案昵称；营销岛传「甲方」类文案） */
  nameA: string
  /** B 方展示名；缺省用 services.hepan.otherPerson */
  nameB?: string
}>()

const { t, te } = useI18n()

const nameB = computed(() => props.nameB || t('services.hepan.otherPerson'))

const PILLAR_LABELS = computed(() => [
  t('profiles.chart.pillar.year'),
  t('profiles.chart.pillar.month'),
  t('profiles.chart.pillar.day'),
  t('profiles.chart.pillar.hour'),
])
function pillarsOf(chart: BaziChartData | null) {
  if (!chart) return []
  const list = [chart.nian, chart.yue, chart.ri, chart.shi]
  return PILLAR_LABELS.value.map((label, i) => ({
    label,
    gan: list[i]?.gan ?? '',
    zhi: list[i]?.zhi ?? '',
    naYin: list[i]?.na_yin ?? '',
  }))
}
const pillarsA = computed(() => pillarsOf(props.chartA))
const pillarsB = computed(() => pillarsOf(props.chartB))

const ZHU_LABELS = computed<Record<string, string>>(() => ({
  nian: t('profiles.chart.pillar.year'),
  yue: t('profiles.chart.pillar.month'),
  ri: t('profiles.chart.pillar.day'),
  shi: t('profiles.chart.pillar.hour'),
}))

function relText(rel: { relation?: string; detail?: string; type?: string }): string {
  return rel.relation || rel.detail || rel.type || ''
}

const dayMasterRel = computed(() => {
  const pairs = props.bond?.zhu_cross?.pairs ?? []
  const pair = pairs.find(p => p.a_zhu === 'ri' && p.b_zhu === 'ri')
  if (!pair) return ''
  return t('services.hepan.dayMasterValue', {
    a: props.nameA,
    aPillar: `${pair.a_gan}${pair.a_zhi}`,
    bPillar: `${pair.b_gan}${pair.b_zhi}`,
    gan: relText(pair.gan_guan_xi),
    zhi: relText(pair.zhi_guan_xi),
  })
})

const keyRels = computed(() => {
  const pairs = props.bond?.zhu_cross?.pairs ?? []
  const set = new Set<string>()
  for (const p of pairs) {
    // 过滤按 type（六合/三合/六冲/六害/相刑），detail 只做展示（如「辰酉合化金」）。
    if (/六合|三合|六冲|六害|相刑/.test(p.zhi_guan_xi.type)) {
      set.add(`${ZHU_LABELS.value[p.a_zhu] ?? p.a_zhu}·${p.a_gan}${p.a_zhi} — ${ZHU_LABELS.value[p.b_zhu] ?? p.b_zhu}·${p.b_gan}${p.b_zhi}：${relText(p.zhi_guan_xi)}`)
    }
  }
  return [...set].slice(0, 8)
})

const nayinRels = computed(() => {
  const pairs = props.bond?.nayin_cross?.pairs ?? []
  return pairs.filter(p => p.relation && p.relation !== '无').map(p =>
    `${ZHU_LABELS.value[p.a_zhu] ?? p.a_zhu}·${p.a_na_yin} ↔ ${ZHU_LABELS.value[p.b_zhu] ?? p.b_zhu}·${p.b_na_yin}：${p.relation}`
  )
})

const shenshaRels = computed(() => {
  const cross = props.bond?.shensha_cross
  if (!cross) return []
  const rows: string[] = []
  for (const [name, info] of Object.entries(cross)) {
    // 引擎键是固定标识符（tian_yi/kui_gang…），展示走 locale 标签。
    const key = `services.hepan.shensha.${name}`
    const label = te(key) ? t(key) : name
    if (info.a_present_in_b) rows.push(t('services.hepan.aStarIntoB', { name: label, a: props.nameA }))
    if (info.b_present_in_a) rows.push(t('services.hepan.bStarIntoA', { name: label, a: props.nameA }))
  }
  return rows.slice(0, 10)
})

const shiShenRows = computed(() => {
  const cross = props.bond?.shi_shen_cross
  if (!cross) return []
  const rows: string[] = []
  const short: Record<string, string> = {
    nian: t('services.hepan.pillarShort.year'),
    yue: t('services.hepan.pillarShort.month'),
    ri: t('services.hepan.pillarShort.day'),
    shi: t('services.hepan.pillarShort.hour'),
  }
  for (const k of ['nian_gan', 'yue_gan', 'ri_gan', 'shi_gan']) {
    const a = cross.a_to_b?.[k]
    const bb = cross.b_to_a?.[k]
    const label = short[k.replace('_gan', '')] ?? k.replace('_gan', '')
    if (a) rows.push(t('services.hepan.aViewB', { a: props.nameA, pillar: label, value: a }))
    if (bb) rows.push(t('services.hepan.bViewA', { a: props.nameA, pillar: label, value: bb }))
  }
  return rows
})
</script>

<template>
  <div class="bond-panel">
    <div class="charts">
      <div class="chart-row">
        <div class="person">
          {{ nameA }}
        </div>
        <div class="pillars">
          <div
            v-for="p in pillarsA"
            :key="p.label"
            class="pillar"
          >
            <div class="pl">
              {{ p.label }}
            </div>
            <div class="gl font-serif">
              {{ p.gan }}{{ p.zhi }}
            </div>
            <div class="sub">
              {{ p.naYin }}
            </div>
          </div>
        </div>
      </div>
      <div class="chart-row">
        <div class="person">
          {{ nameB }}
        </div>
        <div class="pillars">
          <div
            v-for="p in pillarsB"
            :key="p.label"
            class="pillar"
          >
            <div class="pl">
              {{ p.label }}
            </div>
            <div class="gl font-serif">
              {{ p.gan }}{{ p.zhi }}
            </div>
            <div class="sub">
              {{ p.naYin }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="dayMasterRel"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ t('services.hepan.dayMasterRelation') }}
      </div>
      <div class="rel-text">
        {{ dayMasterRel }}
      </div>
    </div>

    <div
      v-if="keyRels.length"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ t('services.hepan.keyEarthlyBranches') }}
      </div>
      <div class="rels">
        <span
          v-for="r in keyRels"
          :key="r"
          class="rel-chip"
        >{{ r }}</span>
      </div>
    </div>

    <div
      v-if="nayinRels.length"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ t('services.hepan.nayinRelation') }}
      </div>
      <div class="rels">
        <span
          v-for="r in nayinRels"
          :key="r"
          class="rel-chip nayin"
        >{{ r }}</span>
      </div>
    </div>

    <div
      v-if="shenshaRels.length"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ t('services.hepan.shenShaCross') }}
      </div>
      <div class="rels">
        <span
          v-for="r in shenshaRels"
          :key="r"
          class="rel-chip shensha"
        >{{ r }}</span>
      </div>
    </div>

    <div
      v-if="shiShenRows.length"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ t('services.hepan.shiShenCross') }}
      </div>
      <div class="rels">
        <span
          v-for="r in shiShenRows"
          :key="r"
          class="rel-chip"
        >{{ r }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.charts { display: grid; gap: var(--space-3); }
.chart-row { display: flex; align-items: center; gap: var(--space-3); }
.person { flex: 0 0 56px; font-weight: 600; font-size: var(--font-size-body); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pillars { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; flex: 1; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.pillar { background: var(--liki-surface); padding: 8px 4px; text-align: center; }
.pillar .pl { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.pillar .gl { font-size: var(--font-size-title-xs); font-weight: 700; }
.pillar .sub { font-size: var(--font-size-micro); color: var(--liki-text-2); }
.rel-text { font-size: var(--font-size-body); color: var(--liki-ink); }
.rels { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.rel-chip { padding: 4px 10px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); background: var(--liki-surface); font-size: var(--font-size-caption); color: var(--liki-text-2); }
/* 纳音关系用品牌色强调；神煞互看为中性参考信息，沿用命盘 shensha chip 的中性语义。 */
.rel-chip.nayin { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.rel-chip.shensha { border-color: var(--liki-line); background: var(--liki-surface-cream); }
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
  .chart-row { flex-direction: column; align-items: flex-start; gap: 6px; }
  .person { flex: none; }
}
</style>
