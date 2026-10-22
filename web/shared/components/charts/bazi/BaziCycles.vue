<template>
  <template v-if="daYunSteps.length">
    <div class="sec-title">{{ $t('profiles.chart.daYun') }}</div>
    <div
      v-if="daYunNote"
      class="sec-note"
    >
      {{ daYunNote }}
    </div>
    <div class="dayun">
      <div
        v-for="(s, i) in daYunSteps"
        :key="`${s.name}-${i}`"
        :class="['dy', { cur: i === currentStepIndex }]"
      >
        <div class="dy-name font-serif">{{ s.name }}</div>
        <div class="dy-shen">{{ s.shi_shen }}</div>
        <div class="dy-year">{{ s.start_year }}–{{ s.end_year }}</div>
      </div>
    </div>
  </template>
  <template v-if="changSheng.length">
    <div class="sec-title">{{ $t('profiles.chart.changShengStages') }}</div>
    <div class="cs-grid">
      <div
        v-for="c in changSheng"
        :key="c.index"
        :class="['cs-cell', { on: pillarZhi.set.has(c.index), cur: c.index === pillarZhi.month }]"
      >
        <div class="cs-zhi">{{ c.index }}</div>
        <div class="cs-name">{{ c.name }}</div>
      </div>
    </div>
  </template>
  <template v-if="nayinRows.length">
    <div class="sec-title">{{ $t('profiles.chart.nayinRel') }}</div>
    <div class="nayin-rel">
      <span
        v-for="(r, i) in nayinRows"
        :key="i"
        class="chart-chip"
      >
        <span class="nr-pos">{{ r.pos }}</span>
        <span class="nr-rel">{{ r.text }}</span>
      </span>
    </div>
  </template>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziChartData, BaziFullChartData } from '../../../api/engine/bazi'

const props = defineProps<{
  chart: BaziChartData | null
  full?: BaziFullChartData | null
}>()

const { t } = useI18n()
const DAYUN_DIR_KEYS: Record<string, string> = {
  顺排: 'profiles.chart.daYunDir.forward',
  逆排: 'profiles.chart.daYunDir.backward',
}
const NAYIN_KEYS: Record<string, string> = {
  相生: 'profiles.chart.nayinRelValue.sheng',
  相克: 'profiles.chart.nayinRelValue.ke',
  相同: 'profiles.chart.nayinRelValue.same',
}
const PILLAR_KEYS = ['year', 'month', 'day', 'hour'] as const
const ENGINE_PILLAR_INDEX: Record<string, number> = { nian: 0, yue: 1, ri: 2, shi: 3 }

function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}
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

const daYunSteps = computed(() => props.chart?.da_yun?.steps ?? [])
const currentYear = new Date().getFullYear()
const currentStepIndex = computed(() => daYunSteps.value.findIndex(
  (step) => (step.start_year ?? 0) <= currentYear && currentYear <= (step.end_year ?? 0),
))
const daYunNote = computed(() => {
  const dy = props.chart?.da_yun ?? props.full?.da_yun
  const parts: string[] = []
  const dir = mapped(DAYUN_DIR_KEYS, dy?.direction)
  if (dir) parts.push(dir)
  const step = daYunSteps.value[currentStepIndex.value]
  if (step?.end_year) parts.push(t('profiles.chart.daYunNext', { years: step.end_year - currentYear }))
  return parts.join(' · ')
})

const changSheng = computed(() => props.full?.chang_sheng ?? [])
const pillarZhi = computed(() => {
  const src = props.full ?? props.chart
  if (!src) return { set: new Set<string>(), month: '' }
  return {
    set: new Set([src.nian.zhi, src.yue.zhi, src.ri.zhi, src.shi.zhi]),
    month: src.yue.zhi,
  }
})

const nayinRows = computed(() => (props.full?.nayin_rel ?? []).map(r => ({
  pos: posText(pillarIndex(r.a), pillarIndex(r.b)),
  text: mapped(NAYIN_KEYS, r.relation),
})))
</script>


<style scoped>
.sec-note { font-size: var(--font-size-caption); color: var(--liki-text-3); margin: calc(-1 * var(--space-1)) 0 var(--space-2); }
.dy.cur .dy-name, .dy.cur .dy-shen { color: var(--liki-primary); }
.cs-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(48px, 1fr)); gap: var(--space-2); }
.cs-cell { border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: 6px 2px; text-align: center; }
.cs-cell.on { background: var(--liki-surface-cream); }
.cs-cell.cur { background: var(--liki-primary-tint); border-color: var(--liki-primary-border); }
.cs-zhi { font-size: var(--font-size-caption); font-weight: 600; }
.cs-cell.cur .cs-zhi { color: var(--liki-primary); }
.cs-name { font-size: var(--font-size-micro); color: var(--liki-text-3); margin-top: 2px; }
.nayin-rel { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.nr-pos { color: var(--liki-text-3); }
.nr-rel { color: var(--liki-ink); }
</style>
