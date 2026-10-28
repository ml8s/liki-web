<template>
  <div
    v-if="isReadyArtifact(descriptor)"
    class="artifact-panel"
    :aria-label="$t('artifact.documentLabel')"
  >
    <template v-if="descriptor.kind === 'ziwei'">
      <ZiweiChartPanel
        :result="descriptor.payload.chart"
      />
      <ZiweiDaXianPanel
        v-if="descriptor.payload.daXian?.length"
        :result="descriptor.payload.daXian"
      />
    </template>

    <template v-else-if="descriptor.kind === 'bazi'">
      <BaziFullChartPanel
        v-if="descriptor.payload.full"
        :result="descriptor.payload.full"
      />
      <BaziChartPanel
        v-else
        :result="descriptor.payload.chart"
      />
      <BaziLiuNianPanel
        v-if="descriptor.payload.liunian"
        :result="descriptor.payload.liunian"
      />
    </template>

    <template v-else-if="descriptor.kind === 'hepan'">
      <BaziBondPanel
        :result="descriptor.payload.bond"
        :chart-a="descriptor.payload.chartA"
        :chart-b="descriptor.payload.chartB"
        :name-a="descriptor.payload.nameA"
        :name-b="descriptor.payload.nameB"
      />
    </template>

    <template v-else-if="descriptor.kind === 'liuyao'">
      <LiuyaoChartPanel
        v-if="descriptor.payload.chart"
        :result="descriptor.payload.chart"
      />
      <LiuyaoCastingPanel
        v-else
        :result="descriptor.payload.casting"
      />
    </template>

    <template v-else-if="descriptor.kind === 'huangli'">
      <HuangliDaysPanel
        :result="descriptor.payload.days"
      />
    </template>

    <div
      v-else-if="descriptor.kind === 'fengshui'"
      class="artifact-fengshui"
    >
      <BazhaiChartPanel
        v-if="descriptor.payload.bazhai"
        :result="descriptor.payload.bazhai"
        :layout="descriptor.payload.layout"
      />
      <BazhaiLayoutPanel
        v-if="descriptor.payload.layout"
        :result="descriptor.payload.layout"
      />
      <XuankongChartPanel
        v-if="descriptor.payload.xuankong"
        :result="descriptor.payload.xuankong"
      />
    </div>

    <BaziYongShenPanel
      v-else-if="descriptor.kind === 'naming'"
      :result="descriptor.payload.assessment"
    />
  </div>

  <div
    v-else
    class="artifact-panel artifact-panel--empty"
    role="status"
  >
    <p>{{ $t('artifact.unavailable') }}</p>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import type { AnyArtifactDescriptor } from './types'
import { isReadyArtifact } from './types'

// Engine panels are lazy: only one renders at a time and only after the user
// expands an artifact capsule. Keeping them async keeps ~50 kB of chart
// implementations out of the initial consultation route graph.
const BazhaiChartPanel = defineAsyncComponent(() => import('@shared/components/engine/fengshui/BazhaiChartPanel.vue'))
const BazhaiLayoutPanel = defineAsyncComponent(() => import('@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'))
const XuankongChartPanel = defineAsyncComponent(() => import('@shared/components/engine/fengshui/XuankongChartPanel.vue'))
const BaziBondPanel = defineAsyncComponent(() => import('@shared/components/engine/bazi/BaziBondPanel.vue'))
const BaziChartPanel = defineAsyncComponent(() => import('@shared/components/engine/bazi/BaziChartPanel.vue'))
const BaziFullChartPanel = defineAsyncComponent(() => import('@shared/components/engine/bazi/BaziFullChartPanel.vue'))
const BaziLiuNianPanel = defineAsyncComponent(() => import('@shared/components/engine/bazi/BaziLiuNianPanel.vue'))
const BaziYongShenPanel = defineAsyncComponent(() => import('@shared/components/engine/bazi/BaziYongShenPanel.vue'))
const HuangliDaysPanel = defineAsyncComponent(() => import('@shared/components/engine/huangli/HuangliDaysPanel.vue'))
const LiuyaoCastingPanel = defineAsyncComponent(() => import('@shared/components/engine/liuyao/LiuyaoCastingPanel.vue'))
const LiuyaoChartPanel = defineAsyncComponent(() => import('@shared/components/engine/liuyao/LiuyaoChartPanel.vue'))
const ZiweiChartPanel = defineAsyncComponent(() => import('@shared/components/engine/ziwei/ZiweiChartPanel.vue'))
const ZiweiDaXianPanel = defineAsyncComponent(() => import('@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'))

defineProps<{
  descriptor: AnyArtifactDescriptor
}>()
</script>

<style scoped>
.artifact-panel {
  min-width: 0;
}
.artifact-panel :deep(.chart-panel),
.artifact-panel :deep([class*='panel']) {
  max-width: 100%;
}
.artifact-panel--empty p {
  margin: 0;
  color: var(--liki-text-2);
}
.artifact-fengshui {
  display: grid;
  gap: var(--space-4);
}
.artifact-panel > .engine-bazi-chart-panel + .engine-bazi-liunian-panel,
.artifact-panel > .engine-ziwei-panel + .engine-ziwei-daxian-panel,
.artifact-fengshui > * + * {
  margin-top: var(--space-4);
}
</style>
