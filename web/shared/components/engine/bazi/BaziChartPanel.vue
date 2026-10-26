<template>
  <div :class="panelClasses">
    <div class="pillars">
      <div
        v-for="item in pillars"
        :key="item.id"
        :class="['pillar', { self: item.id === 'ri', focused: item.id === focused }]"
        :tabindex="-1"
        :aria-label="`${item.label} ${item.gan}${item.zhi}`"
        :aria-current="item.id === focused ? 'true' : undefined"
        :data-focused="item.id === focused ? 'true' : 'false'"
      >
        <div class="label">
          {{ item.label }}
        </div>
        <div class="gl font-serif">
          {{ item.gan }}{{ item.zhi }}
        </div>
        <div class="sub">
          {{ item.shiShen }}
        </div>
        <div class="sub nayin">
          {{ item.naYin }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziChartPanelProps } from '../contract'

const props = defineProps<BaziChartPanelProps>()

const { t } = useI18n()

const panelClasses = computed(() => [
  'engine-bazi-chart-panel',
  `engine-bazi-chart-panel--${props.density ?? 'full'}`,
])

const pillars = computed(() => {
  const source = props.result
  const map = (
    id: 'nian' | 'yue' | 'ri' | 'shi',
    key: 'year' | 'month' | 'day' | 'hour',
    pillar: BaziChartPanelProps['result']['nian'],
  ) => ({
    id,
    label: t(`profiles.chart.pillar.${key}`),
    gan: pillar.gan,
    zhi: pillar.zhi,
    naYin: pillar.na_yin ?? '',
    shiShen: '',
  })

  return [
    map('nian', 'year', source.nian),
    map('yue', 'month', source.yue),
    map('ri', 'day', source.ri),
    map('shi', 'hour', source.shi),
  ]
})
</script>

<style scoped>
.engine-bazi-chart-panel { min-width: 0; }
.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: var(--radius-md); overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: var(--space-4) var(--space-2) var(--space-3); text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.pillar.focused { outline: 2px solid var(--liki-primary); outline-offset: -2px; }
.pillar .label { font-size: var(--font-size-caption); color: var(--liki-text-3); margin-bottom: var(--space-2); letter-spacing: 0.08em; }
.pillar .gl { font-size: var(--font-size-title-lg); font-weight: 700; letter-spacing: 0.02em; }
.pillar.self .gl { color: var(--liki-primary); }
.pillar .sub { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.pillar .nayin { color: var(--liki-text-3); }

@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
}
</style>
