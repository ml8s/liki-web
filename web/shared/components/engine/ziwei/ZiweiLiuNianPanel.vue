<template>
  <section
    :class="['engine-ziwei-liunian-panel', density ? `engine-panel--${density}` : '']"
    :aria-label="$t('profiles.chart.liunian')"
  >
    <div class="sec-title">
      {{ $t('profiles.chart.liunian') }}
    </div>

    <div class="zwc-sihua zwc-sihua--start">
      <span
        v-for="(hua, star) in result.si_hua ?? {}"
        :key="star"
        class="zw-sihua-chip"
      >{{ star }}<i
        class="zw-si"
        :data-hua="hua"
      >{{ hua }}</i>{{ result.si_hua_gong?.[star] ? ` ${result.si_hua_gong[star]}` : '' }}</span>
    </div>

    <div
      class="zw-grid zw-grid--liunian"
      role="group"
      :aria-label="$t('profiles.chart.liunian')"
    >
      <div
        v-for="palace in placedPalaces"
        :id="`ziwei-liunian-${palace.id}`"
        :key="palace.id"
        tabindex="0"
        :style="palace.style"
        :class="['zw-palace', {
          ming: palace.ming,
          focused: palace.focused,
        }]"
        :aria-label="palace.label"
        :aria-current="palace.focused ? 'true' : undefined"
      >
        <div class="zw-pname">
          {{ palace.name }}
        </div>
        <div
          v-if="palace.stars.length"
          class="zw-main"
        >
          {{ palace.stars.join(' ') }}
        </div>
      </div>
      <div class="zw-center">
        <div class="zwc-title">
          {{ $t('profiles.chart.liunian') }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.liuNianPalace', { name: result.ming_gong_name ?? result.ming_gong }) }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import '@shared/components/charts/ziwei-panel.css'
import { computed } from 'vue'
import type { ZiweiLiuNianPanelProps } from '../contract'

const props = defineProps<ZiweiLiuNianPanelProps>()

// Annual placement source: web/app/src/components/birth-profiles/ChartPreview.vue.
const ZHI_POS: Record<string, string> = {
  巳: 'grid-row:1;grid-column:1',
  午: 'grid-row:1;grid-column:2',
  未: 'grid-row:1;grid-column:3',
  申: 'grid-row:1;grid-column:4',
  辰: 'grid-row:2;grid-column:1',
  酉: 'grid-row:2;grid-column:4',
  卯: 'grid-row:3;grid-column:1',
  戌: 'grid-row:3;grid-column:4',
  寅: 'grid-row:4;grid-column:1',
  丑: 'grid-row:4;grid-column:2',
  子: 'grid-row:4;grid-column:3',
  亥: 'grid-row:4;grid-column:4',
}

const placedPalaces = computed(() => (props.result.gong_wei ?? []).map(g => {
  const id = `${g.name}-${g.zhi}`

  return {
    id,
    name: g.name,
    zhi: g.zhi,
    ming: !!g.is_liu_ming,
    focused: props.focused === g.name || props.focused === id,
    stars: g.xing_yao ?? [],
    style: ZHI_POS[g.zhi ?? ''] ?? '',
    label: [g.name, g.zhi].filter(Boolean).join(' '),
  }
}))
</script>

<style scoped>
.engine-ziwei-liunian-panel {
  min-width: 0;
  max-width: 100%;
}

.engine-panel--compact .zw-grid--liunian {
  grid-template-rows: repeat(4, minmax(48px, auto));
}

.zw-palace:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: -2px;
}

.zw-palace.focused {
  box-shadow: inset 0 0 0 2px var(--liki-primary);
}

.zw-pname,
.zw-main,
.zwc-row,
.zw-sihua-chip {
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;
  white-space: normal;
}

@media (max-width: 320px) {
  .zw-grid--liunian {
    grid-template-rows: repeat(4, minmax(44px, auto));
  }

  .zw-palace {
    padding: 3px 2px;
  }
}
</style>
