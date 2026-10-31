<template>
  <div
    class="engine-yongshen"
    :class="density === 'compact' ? 'engine-yongshen--compact' : 'engine-yongshen--full'"
    :data-status="result.status"
    :data-focused="focused"
  >
    <span class="engine-yongshen__status flow-visually-hidden">
      {{ result.status }}
    </span>

    <div
      class="ys-main"
      :class="{ 'is-focused': focused === 'yong' }"
      :aria-current="focused === 'yong' ? 'true' : undefined"
    >
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
          v-for="element in result.selected.xi"
          :key="`xi-${element}`"
          class="ys-chip good"
          :class="{ 'is-focused': focused === element }"
          :aria-current="focused === element ? 'true' : undefined"
        >{{ element }}</span>
      </span>
    </div>

    <div
      v-if="result.selected.ji?.length"
      class="ys-row"
    >
      <span class="ys-label">{{ $t('services.naming.jiShen') }}</span>
      <span class="ys-chips">
        <span
          v-for="element in result.selected.ji"
          :key="`ji-${element}`"
          class="ys-chip bad"
          :class="{ 'is-focused': focused === element }"
          :aria-current="focused === element ? 'true' : undefined"
        >{{ element }}</span>
      </span>
    </div>

    <ul
      v-if="evidenceItems.length"
      class="ys-evidence"
    >
      <li
        v-for="(item, index) in evidenceItems"
        :key="`${item.source}-${index}`"
        class="ys-evidence__row"
        :class="{ 'is-focused': focused === item.source }"
        :aria-current="focused === item.source ? 'true' : undefined"
      >
        <span class="ys-label">{{ item.source }}</span>
        <span class="ys-evidence__body">{{ evidenceText(item) }}</span>
      </li>
    </ul>

    <ul
      v-if="conflictItems.length"
      class="ys-conflicts"
    >
      <li
        v-for="conflict in conflictItems"
        :key="conflict.code"
        class="ys-conflict"
        :data-code="conflict.code"
        :class="{ 'is-focused': focused === conflict.code }"
        :aria-current="focused === conflict.code ? 'true' : undefined"
      >
        {{ conflict.message }}
      </li>
    </ul>

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
import type { BaziYongShenPanelProps } from '../contract'

type EvidenceItem = BaziYongShenPanelProps['result']['evidence'][number]

const props = defineProps<BaziYongShenPanelProps>()

const evidenceItems = computed(() => props.result.evidence ?? [])
const conflictItems = computed(() => props.result.conflicts ?? [])

const needsReview = computed(() =>
  props.result.status === 'needs_review'
  || props.result.selected.yong === null
  || conflictItems.value.length > 0,
)

function evidenceValueText(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (Array.isArray(value)) {
    const values = value.map(evidenceValueText).filter(text => text !== '—')
    return values.length ? values.join('、') : '—'
  }
  if (typeof value === 'object') {
    const values = Object.values(value).map(evidenceValueText).filter(text => text !== '—')
    return values.length ? values.join('、') : '—'
  }
  return String(value)
}

function evidenceText(item: EvidenceItem): string {
  return Object.entries(item)
    .filter(([key]) => key !== 'source')
    .map(([, value]) => evidenceValueText(value))
    .join(' · ')
}
</script>

<style scoped>
.engine-yongshen {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  overflow-wrap: anywhere;
}
.engine-yongshen--compact {
  gap: var(--space-2);
}
.flow-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}
.ys-main,
.ys-row {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-3);
}
.engine-yongshen--compact .ys-main,
.engine-yongshen--compact .ys-row {
  gap: var(--space-2);
}
.ys-label {
  min-width: 0;
  flex: 0 0 auto;
  font-size: var(--font-size-label);
  color: var(--liki-text-3);
}
.ys-value {
  min-width: 0;
  font-size: var(--font-size-title-lg);
  font-weight: 700;
  color: var(--liki-primary);
}
.ys-source {
  min-width: 0;
  font-size: var(--font-size-caption);
  color: var(--liki-text-3);
}
.ys-chips,
.ys-evidence,
.ys-conflicts {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}
.ys-chip {
  min-width: 0;
  display: inline-block;
  padding: 2px var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  font-size: var(--font-size-body-lg);
}
.ys-chip.good {
  border-color: var(--color-success-border);
  background: var(--color-success-surface);
  color: var(--color-success);
}
.ys-chip.bad {
  border-color: var(--color-error-border);
  background: var(--color-danger-surface);
  color: var(--color-fire);
}
.ys-evidence__row,
.ys-conflict {
  min-width: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
.ys-evidence__row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
.ys-evidence__body {
  min-width: 0;
}
.ys-review {
  min-width: 0;
  margin: var(--space-1) 0 0;
  font-size: var(--font-size-caption);
  color: var(--liki-text-2);
}
.is-focused {
  border-color: var(--liki-primary-border) !important;
  background: var(--liki-primary-tint) !important;
}
</style>
