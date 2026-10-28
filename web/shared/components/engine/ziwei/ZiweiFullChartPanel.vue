<template>
  <section
    :class="['engine-ziwei-full-panel', density ? `engine-panel--${density}` : '']"
    :aria-label="$t('profiles.chart.ziwei')"
  >
    <ZiweiChartPanel
      :result="result"
      :focused="focused"
      :density="density"
    />

    <div
      v-if="result.patterns?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.patterns') }}
      </div>
      <div class="zw-chips">
        <span
          v-for="pattern in result.patterns"
          :key="pattern.name"
          class="chart-chip"
          :title="pattern.description"
        >{{ pattern.name }}</span>
      </div>
    </div>

    <div
      v-if="result.san_fang?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.sanFang') }}
      </div>
      <div class="zw-sanfang">
        <div
          v-for="sanFang in result.san_fang"
          :key="sanFang.name"
          class="zw-sf"
        >
          <div class="zw-sf-name">
            {{ sanFang.name }}
          </div>
          <div class="zw-sf-body">
            {{ sanFang.zhu_xing.join(' ') }}<template v-if="sanFang.fu_xing?.length">
              · {{ sanFang.fu_xing.join(' ') }}
            </template><i
              v-for="(hua, index) in sanFang.si_hua ?? []"
              :key="index"
              class="zw-si"
              :data-hua="hua"
            >{{ hua }}</i>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="result.kong_gong?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.kongJie') }}
      </div>
      <div class="zw-chips">
        <span
          v-for="kongGong in result.kong_gong"
          :key="kongGong.gong_name"
          class="chart-chip"
        >{{ $t('profiles.chart.kongJieValue', {
          gong: kongGong.gong_name,
          dui: kongGong.dui_gong,
          stars: kongGong.jie_xing.join(' '),
        }) }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import '@shared/components/charts/ziwei-panel.css'
import type { ZiweiFullChartPanelProps } from '../contract'
import ZiweiChartPanel from './ZiweiChartPanel.vue'

defineProps<ZiweiFullChartPanelProps>()
</script>

<style scoped>
.engine-ziwei-full-panel {
  min-width: 0;
  max-width: 100%;
}

.chart-chip,
.zw-sf,
.zw-sf-body {
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>
