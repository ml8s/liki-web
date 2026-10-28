<template>
  <section
    :class="['engine-ziwei-daxian-panel', density ? `engine-panel--${density}` : '']"
    :aria-label="$t('profiles.chart.daxian')"
  >
    <div class="sec-title">
      {{ $t('profiles.chart.daxian') }}
    </div>
    <ul class="dayun">
      <li
        v-for="period in projectedPeriods"
        :id="`ziwei-daxian-${period.id}`"
        :key="period.id"
        tabindex="0"
        :class="['dy', {
          cur: period.current,
          focused: period.focused,
        }]"
        :aria-label="period.label"
        :aria-current="period.current || period.focused ? (period.current ? 'date' : 'true') : undefined"
      >
        <div class="dy-name">
          {{ period.name }}
        </div>
        <div class="dy-shen">
          {{ $t('profiles.chart.ageRange', { from: period.qi_sui, to: period.zhi_sui }) }}
        </div>
        <div class="dy-year">
          {{ period.yearRange }}
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ZiweiDaXianPanelProps } from '../contract'

const props = defineProps<ZiweiDaXianPanelProps>()

const { t } = useI18n()

// Source projection: web/shared/components/charts/ZiweiPanel.vue.
const projectedPeriods = computed(() => {
  const currentYear = new Date().getFullYear()

  return props.result.map(item => {
    const id = `${item.name}-${item.qi_sui}`
    const yearRange = item.start_year != null && item.end_year != null
      ? `${item.start_year}–${item.end_year}`
      : ''
    const current = item.start_year != null
      && item.end_year != null
      && currentYear >= item.start_year
      && currentYear <= item.end_year

    return {
      ...item,
      id,
      yearRange,
      current,
      focused: props.focused === id,
      label: [item.name, t('profiles.chart.ageRange', {
        from: item.qi_sui,
        to: item.zhi_sui,
      }), yearRange].filter(Boolean).join(' '),
    }
  })
})
</script>

<style scoped>
.engine-ziwei-daxian-panel {
  min-width: 0;
  max-width: 100%;
}

.dayun {
  list-style: none;
  margin: 0;
  padding: 0;
}

.dy {
  min-width: 0;
  max-width: 100%;
}

.dy:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: 2px;
}

.dy.focused {
  box-shadow: inset 0 0 0 2px var(--liki-primary);
}

.dy-name,
.dy-shen,
.dy-year {
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>
