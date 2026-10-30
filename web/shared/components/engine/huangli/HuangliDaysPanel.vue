<template>
  <div class="huangli-panel">
    <div
      v-if="rangeInfo"
      class="range-info"
    >
      {{ $t('services.huangli.range', { start: rangeInfo.start, end: rangeInfo.end, days: rangeInfo.days }) }}
    </div>
    <div
      v-if="recommended.length"
      class="sec-title"
    >
      {{ $t('services.huangli.recommended') }}
    </div>
    <div
      v-if="recommended.length"
      class="days"
    >
      <HuangliDayCard
        v-for="d in recommended"
        :key="d.date"
        :day="d"
        variant="good"
        @select="emit('select', $event)"
      />
    </div>
    <p
      v-if="!recommended.length"
      class="hint"
    >
      {{ $t('services.huangli.noRecommended') }}
    </p>
    <div
      v-if="possible.length"
      class="sec-title"
    >
      {{ $t('services.huangli.possible') }}
    </div>
    <div
      v-if="possible.length"
      class="days"
    >
      <HuangliDayCard
        v-for="d in possible"
        :key="d.date"
        :day="d"
        variant="possible"
        @select="emit('select', $event)"
      />
    </div>
    <div
      v-if="unsuitable.length"
      class="sec-title"
    >
      {{ $t('services.huangli.unsuitable') }}
    </div>
    <div
      v-if="unsuitable.length"
      class="days"
    >
      <HuangliDayCard
        v-for="d in unsuitable"
        :key="d.date"
        :day="d"
        variant="bad"
        @select="emit('select', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 黄历择日结果展示（区间信息 + 宜/平/忌三组 + 每日详情展开）——纯展示、数据驱动。
// App 的 HuangliView 与营销站 huangli 岛共用；文案同取 services.huangli.*（共享拷贝见 chart-messages.<locale>.json）。
// Provenance: extracted from web/shared/components/charts/HuangliDaysPanel.vue.
import { computed } from 'vue'
import type { HuangliDaysPanelProps } from '../contract'
import HuangliDayCard from './HuangliDayCard.vue'

const props = defineProps<HuangliDaysPanelProps>()

const emit = defineEmits<{
  select: [date: string]
}>()

const recommended = computed(() => props.result.filter(d => d.suitability === 'recommended'))
const possible = computed(() => props.result.filter(d => d.suitability === 'possible'))
const unsuitable = computed(() => props.result.filter(d => d.suitability === 'unsuitable'))
const rangeInfo = computed(() => {
  const list = props.result
  if (!list.length) return null
  return { start: list[0].date, end: list[list.length - 1].date, days: list.length }
})
</script>

<style scoped>
.range-info { font-size: var(--font-size-label); color: var(--liki-text-2); margin-bottom: var(--space-3); font-variant-numeric: tabular-nums; }
.days { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: var(--space-2); align-items: start; }
.day { border: 1px solid var(--liki-line); border-radius: var(--radius-md); padding: var(--space-3); }
/* 展开的日卡占满整行：详情（十二时辰等）获得完整宽度。 */
.day[open] { grid-column: 1 / -1; }
.day.good { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.day.possible { border-color: var(--liki-line); background: var(--liki-surface); }
.day.bad { border-color: var(--color-error-border); background: var(--color-danger-surface); }
.d-summary { position: relative; display: flex; flex-direction: column; align-items: center; gap: 2px; list-style: none; cursor: pointer; }
.d-summary::-webkit-details-marker { display: none; }
.d-summary:focus-visible { outline: 2px solid var(--liki-primary); outline-offset: 2px; border-radius: var(--radius-small); }
.d-date { font-weight: 700; font-variant-numeric: tabular-nums; }
.d-rizhu { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.d-sub { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.d-caret { position: absolute; right: 0; top: 2px; width: 7px; height: 7px; border-right: 1.5px solid var(--liki-text-3); border-bottom: 1.5px solid var(--liki-text-3); transform: rotate(45deg); transition: transform .2s ease; }
.day[open] .d-caret { transform: rotate(-135deg); }
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: var(--space-1) 0 var(--space-3); }
</style>
