<template>
  <div class="sec-title">
    {{ $t('profiles.chart.annualStars', { year: liuNian.year }) }}
  </div>
  <div class="ln-row">
    {{ $t('profiles.chart.centerStar') }}<strong>{{ liuNian.ru_zhong }}</strong>
  </div>
  <div class="chart-grid">
    <div
      v-for="f in flyStars"
      :key="f.gong_num"
      :class="['fly-cell', { center: f.gong_num === 5, ru_zhong: f.ru_zhong }]"
    >
      <div class="fly-name">
        {{ f.xing_name }}
      </div>
      <div
        v-if="f.wuxing"
        class="fly-wuxing"
      >
        {{ f.wuxing }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { LUOSHU_ORDER } from '../../utils/bazhai'
import type { BazhaiLiuNianXing } from '../../api/engine/fengshui'

interface FlyItem {
  gong_num: number
  xing_name: string
  wuxing?: string
  ru_zhong?: boolean
}

const props = defineProps<{ liuNian: BazhaiLiuNianXing }>()

const flyStars = computed<FlyItem[]>(() => {
  const gong = props.liuNian.gong_wei
  if (!gong || gong.length !== 9) return []
  return LUOSHU_ORDER.map(n => gong.find(g => g.gong_num === n) ?? { gong_num: n, xing_name: '—' })
})
</script>


<style scoped>
.ln-row { font-size: var(--font-size-label); color: var(--liki-text-2); margin-bottom: var(--space-3); }
.fly-cell { background: var(--liki-surface); padding: var(--space-3) var(--space-1); text-align: center; }
.fly-cell.center { background: var(--liki-primary-tint); }
.fly-cell.ru_zhong { border: 2px solid var(--liki-primary); background: var(--liki-primary-tint); }
.fly-name { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.fly-wuxing { font-size: var(--font-size-micro); color: var(--liki-text-3); margin-top: 2px; }
</style>
