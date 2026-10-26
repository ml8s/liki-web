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
import BazhaiChartPanel from '@shared/components/engine/fengshui/BazhaiChartPanel.vue'
import BazhaiLayoutPanel from '@shared/components/engine/fengshui/BazhaiLayoutPanel.vue'
import XuankongChartPanel from '@shared/components/engine/fengshui/XuankongChartPanel.vue'
import BaziBondPanel from '@shared/components/engine/bazi/BaziBondPanel.vue'
import BaziChartPanel from '@shared/components/engine/bazi/BaziChartPanel.vue'
import BaziFullChartPanel from '@shared/components/engine/bazi/BaziFullChartPanel.vue'
import BaziLiuNianPanel from '@shared/components/engine/bazi/BaziLiuNianPanel.vue'
import BaziYongShenPanel from '@shared/components/engine/bazi/BaziYongShenPanel.vue'
import HuangliDaysPanel from '@shared/components/engine/huangli/HuangliDaysPanel.vue'
import LiuyaoCastingPanel from '@shared/components/engine/liuyao/LiuyaoCastingPanel.vue'
import LiuyaoChartPanel from '@shared/components/engine/liuyao/LiuyaoChartPanel.vue'
import ZiweiChartPanel from '@shared/components/engine/ziwei/ZiweiChartPanel.vue'
import ZiweiDaXianPanel from '@shared/components/engine/ziwei/ZiweiDaXianPanel.vue'
import type { AnyArtifactDescriptor } from './types'
import { isReadyArtifact } from './types'

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
