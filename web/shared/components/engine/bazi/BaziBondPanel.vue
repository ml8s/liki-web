<template>
  <div
    class="engine-bond"
    :class="density === 'compact' ? 'engine-bond--compact' : 'engine-bond--full'"
    :data-focused="focused"
  >
    <div class="charts">
      <div
        v-for="chart in charts"
        :key="chart.side"
        class="chart-row"
      >
        <div class="person">
          {{ chart.name }}
        </div>
        <div class="pillars">
          <div
            v-for="pillar in chart.pillars"
            :key="`${chart.side}-${pillar.key}`"
            :data-cell-id="`${chart.side}:${pillar.key}`"
            class="pillar"
            role="group"
            tabindex="0"
            :aria-label="pillarAriaLabel(pillar)"
            :class="{ 'is-focused': focused === `${chart.side}:${pillar.key}` }"
          >
            <div class="pl">
              {{ pillar.label }}
            </div>
            <div class="gl font-serif">
              {{ pillar.gan }}{{ pillar.zhi }}
            </div>
            <div class="sub">
              {{ pillar.naYin }}
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
          v-for="relation in keyRels"
          :key="relation"
          class="rel-chip"
        >{{ relation }}</span>
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
          v-for="relation in nayinRels"
          :key="relation"
          class="rel-chip nayin"
        >{{ relation }}</span>
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
          v-for="relation in shenshaRels"
          :key="relation"
          class="rel-chip shensha"
        >{{ relation }}</span>
      </div>
    </div>

    <div
      v-if="shiShenRows.length"
      class="rel-sec shi-shen"
    >
      <div class="sec-title">
        {{ t('services.hepan.shiShenCross') }}
      </div>
      <div class="rels">
        <span
          v-for="relation in shiShenRows"
          :key="relation"
          class="rel-chip"
        >{{ relation }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 合盘结果盘面（共享展示组件）：两盘四柱对照 + 日主/地支/纳音/神煞/十神互看。
// 文案取自共享消息拷贝（profiles.chart.* / services.hepan.*）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziBondPanelProps } from '../contract'
import type { BaziChartData, BaziZhu } from '../../../api/engine/bazi'

type PillarKey = 'nian' | 'yue' | 'ri' | 'shi'

interface ProjectedPillar {
  key: PillarKey
  label: string
  gan: string
  zhi: string
  naYin: string
}

const props = defineProps<BaziBondPanelProps>()
const { t, te } = useI18n()

const displayNameA = computed(() => props.nameA?.trim() || t('services.hepan.profileA'))
const displayNameB = computed(() => props.nameB?.trim() || t('services.hepan.otherPerson'))

const PILLAR_KEYS: readonly PillarKey[] = ['nian', 'yue', 'ri', 'shi']
const PILLAR_LOCALE_KEYS: Record<PillarKey, 'year' | 'month' | 'day' | 'hour'> = {
  nian: 'year',
  yue: 'month',
  ri: 'day',
  shi: 'hour',
}

const charts = computed(() => [{
  side: 'A' as const,
  name: displayNameA.value,
  pillars: pillarsOf(props.chartA),
}, {
  side: 'B' as const,
  name: displayNameB.value,
  pillars: pillarsOf(props.chartB),
}])

function pillarLabel(key: PillarKey): string {
  return t(`profiles.chart.pillar.${PILLAR_LOCALE_KEYS[key]}`)
}

function pillarsOf(chart: BaziChartData): ProjectedPillar[] {
  return PILLAR_KEYS.map((key) => {
    const zhu = chart?.[key] as BaziZhu | undefined
    return {
      key,
      label: pillarLabel(key),
      gan: zhu?.gan ?? '',
      zhi: zhu?.zhi ?? '',
      naYin: zhu?.na_yin ?? '',
    }
  })
}

function pillarAriaLabel(pillar: ProjectedPillar): string {
  return pillar.naYin
    ? `${pillar.label} ${pillar.gan}${pillar.zhi} ${pillar.naYin}`
    : `${pillar.label} ${pillar.gan}${pillar.zhi}`
}

const zhuLabels = computed<Record<PillarKey, string>>(() => ({
  nian: pillarLabel('nian'),
  yue: pillarLabel('yue'),
  ri: pillarLabel('ri'),
  shi: pillarLabel('shi'),
}))

function relationText(relation: { relation?: string; detail?: string; type?: string }): string {
  return relation?.relation || relation?.detail || relation?.type || ''
}

const dayMasterRel = computed(() => {
  const pairs = props.result.zhu_cross?.pairs ?? []
  const pair = pairs.find(item => item.a_zhu === 'ri' && item.b_zhu === 'ri')
  if (!pair) return ''
  return t('services.hepan.dayMasterValue', {
    a: displayNameA.value,
    aPillar: `${pair.a_gan}${pair.a_zhi}`,
    bPillar: `${pair.b_gan}${pair.b_zhi}`,
    gan: relationText(pair.gan_guan_xi),
    zhi: relationText(pair.zhi_guan_xi),
  })
})

const keyRels = computed(() => {
  const pairs = props.result.zhu_cross?.pairs ?? []
  const seen = new Set<string>()
  for (const pair of pairs) {
    // 过滤按 type（六合/三合/六冲/六害/相刑），detail 只做展示（如「辰酉合化金」）。
    if (/六合|三合|六冲|六害|相刑/.test(pair.zhi_guan_xi?.type ?? '')) {
      seen.add(`${zhuLabels.value[pair.a_zhu as PillarKey] ?? pair.a_zhu}·${pair.a_gan}${pair.a_zhi} — ${zhuLabels.value[pair.b_zhu as PillarKey] ?? pair.b_zhu}·${pair.b_gan}${pair.b_zhi}：${relationText(pair.zhi_guan_xi)}`)
    }
  }
  return [...seen].slice(0, 8)
})

const nayinRels = computed(() => {
  const pairs = props.result.nayin_cross?.pairs ?? []
  return pairs
    .filter(pair => pair.relation && pair.relation !== '无')
    .map(pair => `${zhuLabels.value[pair.a_zhu as PillarKey] ?? pair.a_zhu}·${pair.a_na_yin} ↔ ${zhuLabels.value[pair.b_zhu as PillarKey] ?? pair.b_zhu}·${pair.b_na_yin}：${pair.relation}`)
})

const shenshaRels = computed(() => {
  const cross = props.result.shensha_cross
  if (!cross) return []
  const rows: string[] = []
  for (const [name, info] of Object.entries(cross)) {
    // 引擎键是固定标识符（tian_yi/kui_gang…），展示走 locale 标签。
    const key = `services.hepan.shensha.${name}`
    const label = te(key) ? t(key) : name
    if (info?.a_present_in_b) {
      rows.push(t('services.hepan.aStarIntoB', { name: label, a: displayNameA.value }))
    }
    if (info?.b_present_in_a) {
      rows.push(t('services.hepan.bStarIntoA', { name: label, a: displayNameA.value }))
    }
  }
  return rows.slice(0, 10)
})

const shiShenRows = computed(() => {
  const cross = props.result.shi_shen_cross
  if (!cross) return []
  const rows: string[] = []
  const shortLabels: Record<PillarKey, string> = {
    nian: t('services.hepan.pillarShort.year'),
    yue: t('services.hepan.pillarShort.month'),
    ri: t('services.hepan.pillarShort.day'),
    shi: t('services.hepan.pillarShort.hour'),
  }
  for (const pillarKey of PILLAR_KEYS) {
    const key = `${pillarKey}_gan` as 'nian_gan' | 'yue_gan' | 'ri_gan' | 'shi_gan'
    const aToB = cross.a_to_b?.[key]
    const bToA = cross.b_to_a?.[key]
    if (aToB) {
      rows.push(t('services.hepan.aViewB', { a: displayNameA.value, pillar: shortLabels[pillarKey], value: aToB }))
    }
    if (bToA) {
      rows.push(t('services.hepan.bViewA', { a: displayNameA.value, pillar: shortLabels[pillarKey], value: bToA }))
    }
  }
  return rows
})
</script>

<style scoped>
.engine-bond {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--liki-ink);
}
.charts {
  min-width: 0;
  display: grid;
  gap: var(--space-3);
}
.engine-bond--compact .charts {
  gap: var(--space-2);
}
.chart-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.person {
  flex: 0 0 56px;
  min-width: 0;
  overflow-wrap: anywhere;
  font-weight: 600;
  font-size: var(--font-size-body);
}
.pillars {
  min-width: 0;
  flex: 1;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  background: var(--liki-line);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-small);
  overflow: hidden;
}
.pillar {
  min-width: 0;
  padding: 8px 4px;
  background: var(--liki-surface);
  text-align: center;
}
.engine-bond--compact .pillar {
  padding: 4px 2px;
}
.pillar .pl {
  min-width: 0;
  font-size: var(--font-size-micro);
  color: var(--liki-text-3);
}
.pillar .gl {
  min-width: 0;
  font-size: var(--font-size-title-xs);
  font-weight: 700;
}
.pillar .sub {
  min-width: 0;
  font-size: var(--font-size-micro);
  color: var(--liki-text-2);
}
.rel-sec {
  min-width: 0;
}
.rel-sec + .rel-sec {
  margin-top: var(--space-3);
}
.engine-bond--compact .rel-sec + .rel-sec {
  margin-top: var(--space-2);
}
.rel-text {
  min-width: 0;
  font-size: var(--font-size-body);
}
.rels {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
.rel-chip {
  min-width: 0;
  padding: 4px 10px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  font-size: var(--font-size-caption);
  color: var(--liki-text-2);
}
/* 纳音关系用品牌色强调；神煞互看为中性参考信息，沿用命盘 shensha chip 的中性语义。 */
.rel-chip.nayin {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}
.rel-chip.shensha {
  border-color: var(--liki-line);
  background: var(--liki-surface-cream);
}
.pillar:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: -3px;
}
.pillar.is-focused {
  border-color: var(--liki-primary-border);
  background: var(--liki-primary-tint);
}
@media (max-width: 640px) {
  .pillars {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .chart-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }
  .person {
    flex: none;
  }
}
</style>
