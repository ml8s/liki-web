<template>
  <div class="yongshen-panel">
    <div class="ys-main">
      <span class="ys-label">{{ $t('services.naming.yongShen') }}</span>
      <span class="ys-value font-serif">{{ result.selected.yong || '—' }}</span>
      <span
        v-if="result.selected.source"
        class="ys-source"
      >{{ result.selected.source }}</span>
    </div>
    <div
      v-if="result.selected.xi?.length"
      class="ys-row"
    >
      <span class="ys-label">{{ $t('services.naming.xiShen') }}</span>
      <span class="ys-chips">
        <span
          v-for="w in result.selected.xi"
          :key="w"
          class="ys-chip good"
        >{{ w }}</span>
      </span>
    </div>
    <div
      v-if="result.selected.ji?.length"
      class="ys-row"
    >
      <span class="ys-label">{{ $t('services.naming.jiShen') }}</span>
      <span class="ys-chips">
        <span
          v-for="w in result.selected.ji"
          :key="w"
          class="ys-chip bad"
        >{{ w }}</span>
      </span>
    </div>
    <p
      v-if="needsReview"
      class="ys-review"
      role="note"
    >
      {{ $t('services.naming.needsReview') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 用神裁决展示（用神/喜神/忌神 + 判据 + 复核提示）——纯展示、数据驱动。
// 数据来自 counsel 的 bazi_yongshen（App 与营销站 naming 岛共用）。
import { computed } from 'vue'
import type { BaziYongShenResult } from '../../api/engine/bazi'

const props = defineProps<{ result: BaziYongShenResult }>()

const needsReview = computed(() =>
  props.result.status === 'needs_review'
  || props.result.selected.yong === null
  || props.result.selected.confidence === 'low'
  || !!props.result.conflicts?.length,
)
</script>

<style scoped>
.yongshen-panel { display: flex; flex-direction: column; gap: var(--space-3); }
.ys-main { display: flex; align-items: baseline; gap: var(--space-3); }
.ys-label { font-size: var(--font-size-label); color: var(--liki-text-3); flex: 0 0 2.5em; }
.ys-value { font-size: var(--font-size-title-lg); font-weight: 700; color: var(--liki-primary); }
.ys-source { font-size: var(--font-size-caption); color: var(--liki-text-3); }
.ys-row { display: flex; align-items: baseline; gap: var(--space-3); }
.ys-chips { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.ys-chip {
  display: inline-block; padding: 2px var(--space-3); border-radius: var(--radius-full);
  font-size: var(--font-size-body-lg); border: 1px solid var(--liki-line);
}
.ys-chip.good { border-color: var(--color-success-border); background: var(--color-success-surface); color: var(--color-success); }
.ys-chip.bad { border-color: var(--color-error-border); background: var(--color-danger-surface); color: var(--color-fire); }
.ys-review { margin: var(--space-1) 0 0; font-size: var(--font-size-caption); color: var(--liki-text-2); }
</style>
