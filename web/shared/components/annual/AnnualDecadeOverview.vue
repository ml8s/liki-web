<template>
  <section
    class="annual-decade"
    :aria-label="$t('profiles.chart.annualDecade')"
  >
    <p class="annual-decade__hint">
      {{ $t('profiles.chart.annualDecadeHint') }}
    </p>

    <div
      class="annual-decade__grid"
      role="listbox"
      :aria-label="$t('profiles.chart.annualDecade')"
    >
      <button
        v-for="item in years"
        :key="item.year"
        type="button"
        role="option"
        class="annual-year"
        :class="[`annual-year--${item.availability}`, { 'is-selected': item.year === selectedYear }]"
        :aria-selected="item.year === selectedYear"
        @click="$emit('select', item.year)"
      >
        <span class="annual-year__year font-serif">{{ item.year }}</span>
        <span class="annual-year__system">
          <span class="annual-year__label">{{ $t('profiles.chart.annualBazi') }}</span>
          <span class="annual-year__value">{{ item.bazi?.nian_name ?? $t('profiles.chart.annualMissing') }}</span>
        </span>
        <span class="annual-year__system">
          <span class="annual-year__label">{{ $t('profiles.chart.annualZiwei') }}</span>
          <span class="annual-year__value">{{ item.ziwei?.ming_gong_name ?? item.ziwei?.ming_gong ?? $t('profiles.chart.annualMissing') }}</span>
        </span>
        <span
          class="annual-year__availability"
          :data-availability="item.availability"
        >
          {{ $t(`profiles.chart.annual${item.availability === 'ready' ? 'Ready' : item.availability === 'partial' ? 'Partial' : 'Missing'}`) }}
        </span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// 十年流年概览：八字与紫微各自请求、按年合并；缺失年份显式呈现，不前端伪造。
import type { AnnualDecadeItem } from '../../domain/natal'

defineProps<{
  years: AnnualDecadeItem[]
  selectedYear: number
}>()

defineEmits<{
  select: [year: number]
}>()
</script>

<style scoped>
.annual-decade__hint {
  margin: 0 0 var(--space-3);
  color: var(--liki-text-2);
  font-size: var(--font-size-label);
  line-height: 1.6;
}

.annual-decade__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-2);
}

.annual-year {
  min-width: 0;
  min-height: var(--control-height-touch);
  display: grid;
  gap: var(--space-1);
  padding: var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-md);
  background: var(--liki-surface);
  color: var(--liki-ink);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.annual-year:hover {
  border-color: var(--liki-primary-border);
}

.annual-year:focus-visible {
  outline: 3px solid var(--liki-primary);
  outline-offset: 2px;
}

.annual-year.is-selected {
  border-color: var(--liki-primary);
  background: var(--liki-primary-tint);
  box-shadow: inset 0 0 0 1px var(--liki-primary);
}

.annual-year__year {
  font-size: var(--font-size-title-xs);
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.annual-year__system {
  min-width: 0;
  display: grid;
  gap: 0;
}

.annual-year__label {
  color: var(--liki-text-3);
  font-size: var(--font-size-micro);
}

.annual-year__value {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}

.annual-year__availability {
  margin-top: var(--space-1);
  color: var(--liki-text-3);
  font-size: var(--font-size-micro);
}

.annual-year__availability[data-availability='ready'] {
  color: var(--liki-primary);
}

.annual-year__availability[data-availability='partial'] {
  color: var(--liki-text-2);
}

.annual-year__availability[data-availability='missing'] {
  color: var(--liki-text-3);
}

@media (max-width: 479.98px) {
  .annual-decade__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .annual-year {
    padding: var(--space-2);
  }
}
</style>
