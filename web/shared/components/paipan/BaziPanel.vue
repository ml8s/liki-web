<template>
  <div
    v-if="chart"
    class="bazi-panel"
  >
    <BaziPillars
      :chart="chart"
      :full="full"
      :liunian="liunian"
    />
    <BaziRelations :full="full" />
    <BaziCycles
      :chart="chart"
      :full="full"
    />
    <BaziLiunianDetail
      :chart="chart"
      :full="full"
      :liunian="liunian"
    />
    <BaziYongShen :full="full" />
  </div>
</template>

<script setup lang="ts">
// BaziPanel is now an orchestrator. Each section owns its own projection logic.
import BaziPillars from './bazi/BaziPillars.vue'
import BaziRelations from './bazi/BaziRelations.vue'
import BaziCycles from './bazi/BaziCycles.vue'
import BaziLiunianDetail from './bazi/BaziLiunianDetail.vue'
import BaziYongShen from './bazi/BaziYongShen.vue'
import type { BaziChartData, BaziFullChartData, BaziLiuNianOutput } from '../../api/engine/bazi'

defineProps<{
  chart: BaziChartData | null
  full?: BaziFullChartData | null
  liunian?: BaziLiuNianOutput | null
}>()
</script>

<style src="./bazi/bazi-panel.css"></style>
