<template>
  <section
    :class="['engine-ziwei-panel', density ? `engine-panel--${density}` : '']"
    :aria-label="$t('profiles.chart.ziwei')"
  >
    <div
      class="zw-grid"
      role="group"
      :aria-label="$t('profiles.chart.ziwei')"
    >
      <div
        v-for="palace in placedPalaces"
        :id="`ziwei-palace-${palace.id}`"
        :key="palace.id"
        tabindex="0"
        :style="palace.style"
        :class="['zw-palace', {
          ming: palace.ming,
          shen: palace.shen,
          focused: palace.focused,
        }]"
        :aria-label="palace.label"
        :aria-current="palace.focused ? 'true' : undefined"
        :title="palace.tooltip"
      >
        <div class="zw-head">
          <span class="zw-pname">{{ palace.name }}</span>
          <span class="zw-gz font-serif">{{ palace.gan }}{{ palace.zhi }}</span>
        </div>
        <div
          v-if="palace.main.length"
          class="zw-main"
        >
          <span
            v-for="star in palace.main"
            :key="star.name"
            class="zw-star"
          >{{ star.name }}{{ star.liang_du }}<i
            v-if="star.si_hua"
            class="zw-si"
            :data-hua="star.si_hua"
          >{{ star.si_hua }}</i></span>
        </div>
        <div
          v-if="palace.minor.length"
          class="zw-minor"
        >
          <span
            v-for="star in palace.minor"
            :key="star.name"
            class="zw-star"
          >{{ star.name }}{{ star.liang_du }}<i
            v-if="star.si_hua"
            class="zw-si"
            :data-hua="star.si_hua"
          >{{ star.si_hua }}</i></span>
        </div>
        <div
          v-if="palace.zaYao.length"
          class="zw-za"
        >
          {{ palace.zaYao.join(' ') }}
        </div>
        <div class="zw-foot">
          <span>{{ $t('profiles.chart.changSheng') }} {{ palace.changSheng }}</span>
          <b
            v-if="palace.shen"
            class="zw-shen"
          >身</b>
        </div>
      </div>
      <div class="zw-center">
        <div class="zwc-title">
          {{ result.ju_shu }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.mingZhu') }} {{ result.ming_zhu }} · {{ $t('profiles.chart.shenZhu') }} {{ result.shen_zhu }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.ziweiPos') }} {{ result.ziwei_pos }} · {{ $t('profiles.chart.shenGong') }} {{ result.shen_gong }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.ziweiBirth', { nian: `${result.nian_gan}${result.nian_zhi}`, shi: result.shi_zhi }) }}
        </div>
        <div class="zwc-sihua">
          <span
            v-for="(hua, star) in result.si_hua ?? {}"
            :key="star"
            class="zw-sihua-chip"
          >{{ star }}<i
            class="zw-si"
            :data-hua="hua"
          >{{ hua }}</i></span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import '@shared/components/charts/ziwei-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ZiweiStarInfo } from '../../../api/engine/ziwei'
import type { ZiweiChartPanelProps } from '../contract'

const props = defineProps<ZiweiChartPanelProps>()

const { t } = useI18n()

// Source order/placement: web/shared/components/charts/ZiweiPanel.vue.
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

interface ZiweiPalaceVM {
  id: string
  name: string
  gan: string
  zhi: string
  ming: boolean
  shen: boolean
  focused: boolean
  style: string
  main: ZiweiStarInfo[]
  minor: ZiweiStarInfo[]
  zaYao: string[]
  changSheng: string
  label: string
  tooltip: string
}

const placedPalaces = computed<ZiweiPalaceVM[]>(() => props.result.gong_wei.map(g => {
  const ages = (g.ages ?? []).join('、')
  const ganZhi = `${g.gan ?? ''}${g.zhi ?? ''}`
  const id = g.name || g.index

  return {
    id,
    name: g.name,
    gan: g.gan ?? '',
    zhi: g.zhi ?? '',
    ming: g.name === props.result.ming_gong,
    shen: !!g.is_shen_gong,
    focused: id !== '' && props.focused === id,
    style: ZHI_POS[g.zhi ?? ''] ?? '',
    main: (g.xing_yao ?? []).filter(star => star.is_zhu_xing),
    minor: (g.xing_yao ?? []).filter(star => !star.is_zhu_xing),
    zaYao: g.za_yao ?? [],
    changSheng: g.chang_sheng ?? '',
    label: [g.name, ganZhi].filter(Boolean).join(' '),
    tooltip: ages ? t('profiles.chart.xiaoXian', { ages }) : '',
  }
}))
</script>

<style scoped>
.engine-ziwei-panel {
  min-width: 0;
  max-width: 100%;
}

.engine-panel--compact .zw-grid {
  grid-template-rows: repeat(4, minmax(76px, auto));
}

.zw-palace:focus-visible {
  outline: 2px solid var(--liki-primary);
  outline-offset: -2px;
}

.zw-palace.focused {
  box-shadow: inset 0 0 0 2px var(--liki-primary);
}

.zw-pname,
.zw-star,
.zwc-row {
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;
  white-space: normal;
}

@media (max-width: 320px) {
  .zw-grid {
    grid-template-rows: repeat(4, minmax(72px, auto));
  }

  .zw-palace {
    padding: 3px 2px;
  }
}
</style>
