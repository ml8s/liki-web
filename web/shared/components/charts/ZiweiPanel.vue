<template>
  <div
    v-if="chart"
    class="ziwei-panel"
  >
    <div class="zw-grid">
      <div
        v-for="g in ziweiPlaced"
        :key="g.name"
        :style="g.style"
        :class="['zw-palace', { ming: g.ming, shen: g.shen }]"
        :title="g.tooltip"
      >
        <div class="zw-head">
          <span class="zw-pname">{{ g.name }}</span>
          <span class="zw-gz font-serif">{{ g.gan }}{{ g.zhi }}</span>
        </div>
        <div
          v-if="g.main.length"
          class="zw-main"
        >
          <span
            v-for="s in g.main"
            :key="s.name"
            class="zw-star"
          >{{ s.name }}{{ s.liang_du }}<i
            v-if="s.si_hua"
            class="zw-si"
            :data-hua="s.si_hua"
          >{{ s.si_hua }}</i></span>
        </div>
        <div
          v-if="g.minor.length"
          class="zw-minor"
        >
          <span
            v-for="s in g.minor"
            :key="s.name"
            class="zw-star"
          >{{ s.name }}{{ s.liang_du }}<i
            v-if="s.si_hua"
            class="zw-si"
            :data-hua="s.si_hua"
          >{{ s.si_hua }}</i></span>
        </div>
        <div
          v-if="g.zaYao.length"
          class="zw-za"
        >
          {{ g.zaYao.join(' ') }}
        </div>
        <div class="zw-foot">
          <span>{{ $t('profiles.chart.changSheng') }} {{ g.changSheng }}</span>
          <b
            v-if="g.shen"
            class="zw-shen"
          >身</b>
        </div>
      </div>
      <div class="zw-center">
        <div class="zwc-title">
          {{ chart.ju_shu }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.mingZhu') }} {{ chart.ming_zhu }} · {{ $t('profiles.chart.shenZhu') }} {{ chart.shen_zhu }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.ziweiPos') }} {{ chart.ziwei_pos }} · {{ $t('profiles.chart.shenGong') }} {{ chart.shen_gong }}
        </div>
        <div class="zwc-row">
          {{ $t('profiles.chart.ziweiBirth', { nian: `${chart.nian_gan}${chart.nian_zhi}`, shi: chart.shi_zhi }) }}
        </div>
        <div class="zwc-sihua">
          <span
            v-for="(hua, star) in chart.si_hua ?? {}"
            :key="star"
            class="zw-sihua-chip"
          >{{ star }}<i
            class="zw-si"
            :data-hua="hua"
          >{{ hua }}</i></span>
        </div>
      </div>
    </div>
    <div
      v-if="chart.patterns?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.patterns') }}
      </div>
      <div class="zw-chips">
        <span
          v-for="p in chart.patterns"
          :key="p.name"
          class="chart-chip"
          :title="p.description"
        >{{ p.name }}</span>
      </div>
    </div>
    <div
      v-if="chart.san_fang?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.sanFang') }}
      </div>
      <div class="zw-sanfang">
        <div
          v-for="s in chart.san_fang"
          :key="s.name"
          class="zw-sf"
        >
          <div class="zw-sf-name">
            {{ s.name }}
          </div>
          <div class="zw-sf-body">
            {{ s.zhu_xing.join(' ') }}<template v-if="s.fu_xing?.length">
              · {{ s.fu_xing.join(' ') }}
            </template><i
              v-for="(hua, i) in s.si_hua ?? []"
              :key="i"
              class="zw-si"
              :data-hua="hua"
            >{{ hua }}</i>
          </div>
        </div>
      </div>
    </div>
    <div
      v-if="chart.kong_gong?.length"
      class="zw-section"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.kongJie') }}
      </div>
      <div class="zw-chips">
        <span
          v-for="k in chart.kong_gong"
          :key="k.gong_name"
          class="chart-chip"
        >{{ $t('profiles.chart.kongJieValue', { gong: k.gong_name, dui: k.dui_gong, stars: k.jie_xing.join(' ') }) }}</span>
      </div>
    </div>
    <div
      v-if="daXian.length"
      class="zw-section zw-daxian"
    >
      <div class="sec-title">
        {{ $t('profiles.chart.daxian') }}
      </div>
      <div class="dayun">
        <div
          v-for="d in daXian"
          :key="`${d.name}-${d.qi_sui}`"
          :class="['dy', { cur: d.current }]"
        >
          <div class="dy-name">
            {{ d.name }}
          </div>
          <div class="dy-shen">
            {{ $t('profiles.chart.ageRange', { from: d.qi_sui, to: d.zhi_sui }) }}
          </div>
          <div class="dy-year">
            {{ d.yearRange }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import './ziwei-panel.css'
// 紫微本命盘展示（十二宫格/格局/三方/空劫/大运）——纯展示、数据驱动。
// 流年区块（含年份选择）由宿主编排器自行组合（App 的 ChartPreview）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ZiweiChart, ZiweiDaXianItem, ZiweiStarInfo } from '../../api/engine/ziwei'

const props = defineProps<{
  chart: ZiweiChart
  daXian?: ZiweiDaXianItem[]
}>()

const { t } = useI18n()

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
  name: string
  gan: string
  zhi: string
  ming: boolean
  shen: boolean
  style: string
  main: ZiweiStarInfo[]
  minor: ZiweiStarInfo[]
  zaYao: string[]
  changSheng: string
  tooltip: string
}

const ziweiPlaced = computed<ZiweiPalaceVM[]>(() => {
  const chart = props.chart
  if (!chart) return []
  return (chart.gong_wei ?? []).map(g => {
    const ages = (g.ages ?? []).join('、')
    return {
      name: g.name,
      gan: g.gan ?? '',
      zhi: g.zhi ?? '',
      ming: g.name === chart.ming_gong,
      shen: !!g.is_shen_gong,
      style: ZHI_POS[g.zhi ?? ''] ?? '',
      main: (g.xing_yao ?? []).filter(s => s.is_zhu_xing),
      minor: (g.xing_yao ?? []).filter(s => !s.is_zhu_xing),
      zaYao: g.za_yao ?? [],
      changSheng: g.chang_sheng ?? '',
      tooltip: ages ? t('profiles.chart.xiaoXian', { ages }) : '',
    }
  })
})

const daXian = computed(() => {
  const cy = new Date().getFullYear()
  return (props.daXian ?? []).map(d => ({
    ...d,
    yearRange: d.start_year != null && d.end_year != null ? `${d.start_year}–${d.end_year}` : '',
    current: d.start_year != null && d.end_year != null && cy >= d.start_year && cy <= d.end_year,
  }))
})
</script>
