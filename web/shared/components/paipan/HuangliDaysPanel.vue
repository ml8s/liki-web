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
      <details
        v-for="d in recommended"
        :key="d.date"
        class="day good"
      >
        <summary class="d-summary">
          <span
            class="d-caret"
            aria-hidden="true"
          />
          <span class="d-date">{{ d.date }}</span>
          <span v-if="riZhuText(d)" class="d-rizhu font-serif">{{ riZhuText(d) }}</span>
          <span class="d-sub">{{ subText(d) }}</span>
        </summary>
        <DayDetail :day="d" />
      </details>
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
      <details
        v-for="d in possible"
        :key="d.date"
        class="day possible"
      >
        <summary class="d-summary">
          <span
            class="d-caret"
            aria-hidden="true"
          />
          <span class="d-date">{{ d.date }}</span>
          <span v-if="riZhuText(d)" class="d-rizhu font-serif">{{ riZhuText(d) }}</span>
          <span class="d-sub">{{ subText(d) }}</span>
        </summary>
        <DayDetail :day="d" />
      </details>
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
      <details
        v-for="d in unsuitable"
        :key="d.date"
        class="day bad"
      >
        <summary class="d-summary">
          <span
            class="d-caret"
            aria-hidden="true"
          />
          <span class="d-date">{{ d.date }}</span>
          <span v-if="riZhuText(d)" class="d-rizhu font-serif">{{ riZhuText(d) }}</span>
          <span class="d-sub">{{ subText(d) }}</span>
        </summary>
        <DayDetail :day="d" />
      </details>
    </div>
  </div>
</template>

<script setup lang="ts">
// 黄历择日结果展示（区间信息 + 宜/平/忌三组 + 每日详情展开）——纯展示、数据驱动。
// App 的 HuangliView 与营销站 huangli 岛共用；文案同取 services.huangli.*（共享拷贝见 chart-messages.<locale>.json）。
import { computed } from 'vue'
import type { HuangliDay } from '../../api/engine/huangli'
import DayDetail from './HuangliDayDetail.vue'

const props = defineProps<{ days: HuangliDay[] }>()

const recommended = computed(() => props.days.filter(d => d.suitability === 'recommended'))
const possible = computed(() => props.days.filter(d => d.suitability === 'possible'))
const unsuitable = computed(() => props.days.filter(d => d.suitability === 'unsuitable'))
const rangeInfo = computed(() => {
  const list = props.days
  if (!list.length) return null
  return { start: list[0].date, end: list[list.length - 1].date, days: list.length }
})

function subText(d: HuangliDay): string {
  return d.reason ? `${d.jian_chu} · ${d.reason}` : d.jian_chu
}

function riZhuText(d: HuangliDay): string {
  return d.ri_zhu ? `${d.ri_zhu.gan}${d.ri_zhu.zhi}` : ''
}
</script>

<style scoped src="../../styles/paipan.css"></style>

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
.d-caret { position: absolute; right: 2px; top: 6px; width: 6px; height: 6px; border-right: 1.5px solid var(--liki-text-3); border-bottom: 1.5px solid var(--liki-text-3); transform: rotate(45deg); transition: transform 160ms var(--ease-standard); }
details[open] .d-caret { transform: rotate(-135deg); }
.hint { color: var(--liki-text-2); font-size: var(--font-size-label); margin: var(--space-1) 0 var(--space-3); }
</style>
