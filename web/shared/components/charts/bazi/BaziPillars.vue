<template>
  <div>
    <div :class="['pillars', { 'with-liunian': !!liunian }]">
      <div
        v-for="item in pillars"
        :key="item.label"
        :class="['pillar', { self: item.self }]"
      >
        <div class="label">{{ item.label }}</div>
        <div class="gl font-serif">{{ item.gan }}{{ item.zhi }}</div>
        <div class="sub">{{ item.shiShen }}</div>
        <div class="sub nayin">{{ item.naYin }}</div>
        <div
          v-if="item.cangGan"
          class="sub cang"
        >{{ item.cangGan }}</div>
        <div
          v-if="item.trend || item.selfSitting || item.xunKong"
          class="micro-group"
        >
          <div
            v-if="item.trend"
            class="micro"
          >{{ $t('profiles.chart.starLuck') }} {{ item.trend }}</div>
          <div
            v-if="item.selfSitting"
            class="micro"
          >{{ $t('profiles.chart.selfSitting') }} {{ item.selfSitting }}</div>
          <div
            v-if="item.xunKong"
            class="micro"
          >{{ $t('profiles.chart.xunKong') }} {{ item.xunKong }}</div>
        </div>
        <div
          v-if="item.isVoid || item.isKuiGang || item.selfHeName"
          class="pillar-tags"
        >
          <span
            v-if="item.isVoid"
            class="void-tag"
          >{{ $t('profiles.chart.void') }}</span>
          <span
            v-if="item.isKuiGang"
            class="solid-tag"
          >{{ $t('profiles.chart.kuiGang') }}</span>
          <span
            v-if="item.selfHeName"
            class="line-tag"
          >{{ item.selfHeName }}</span>
        </div>
      </div>
      <div
        v-if="liunian"
        class="pillar liunian"
      >
        <div class="label">{{ $t('profiles.chart.liunianPillar') }} {{ liunian.year }}</div>
        <div class="gl font-serif">{{ liunian.nian_name }}</div>
        <div
          v-if="liunian.shi_shen"
          class="sub"
        >{{ liunian.shi_shen }}</div>
        <div
          v-if="liunian.na_yin"
          class="sub nayin"
        >{{ liunian.na_yin }}</div>
        <div
          v-if="typeof liunian.sheng === 'number' || typeof liunian.ke === 'number'"
          class="micro-group"
        >
          <div class="micro">
            {{ $t('profiles.chart.shengCount') }} {{ liunian.sheng ?? 0 }} · {{ $t('profiles.chart.keCount') }} {{ liunian.ke ?? 0 }}
          </div>
        </div>
      </div>
    </div>
    <div
      v-if="sanYuanText"
      class="san-yuan"
    >
      <span class="sy-label">{{ $t('profiles.chart.threeYuan') }}</span>
      <span class="sy-body">{{ sanYuanText }}</span>
    </div>
    <div
      v-if="shenShaList.length"
      class="shensha"
    >
      <span class="sy-label">{{ $t('profiles.chart.shenSha') }}</span>
      <span
        v-for="s in shenShaList"
        :key="s"
        class="chart-chip"
      >{{ s }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziChartData, BaziFullChartData, BaziLiuNianOutput, BaziZhuFull } from '../../../api/engine/bazi'

const props = defineProps<{
  chart: BaziChartData | null
  full?: BaziFullChartData | null
  liunian?: BaziLiuNianOutput | null
}>()

const { t } = useI18n()


const pillars = computed(() => {
  const src = props.full ?? (props.chart as unknown as { nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull } | null)
  if (!src) return []
  const map = (key: 'year' | 'month' | 'day' | 'hour', p: BaziZhuFull) => {
    const cang = p.cang_gan ? [p.cang_gan.main, p.cang_gan.mid, p.cang_gan.minor].filter(Boolean).join(' ') : ''
    return {
      label: t(`profiles.chart.pillar.${key}`),
      self: key === 'day',
      gan: p.gan,
      zhi: p.zhi,
      naYin: p.na_yin ?? '',
      shiShen: p.shi_shens?.[0]?.shi_shen ?? '',
      cangGan: cang ? t('profiles.chart.hiddenStems', { stems: cang }) : '',
      isVoid: !!p.is_void,
      trend: p.day_master_trend ?? '',
      selfSitting: p.self_sitting ?? '',
      xunKong: p.xun_kong ?? '',
      isKuiGang: !!p.is_kui_gang,
      selfHeName: p.self_he_name ?? '',
    }
  }
  return [map('year', src.nian), map('month', src.yue), map('day', src.ri), map('hour', src.shi)]
})

function zhuText(p?: { gan: string; zhi: string }): string {
  return p ? `${p.gan}${p.zhi}` : ''
}

const sanYuanText = computed(() => {
  const sy = props.full?.san_yuan
  if (!sy) return ''
  return t('profiles.chart.sanYuanValue', {
    taiYuan: zhuText(sy.tai_yuan),
    mingGong: zhuText(sy.ming_gong),
    shenGong: zhuText(sy.shen_gong),
  })
})

const shenShaList = computed(() => {
  const src = props.full
  if (!src) return []
  const names = new Set<string>()
  for (const p of [src.nian, src.yue, src.ri, src.shi]) {
    for (const s of p.shen_sha ?? []) {
      if (s.name) names.add(s.name)
    }
  }
  return [...names]
})
</script>


<style scoped>
.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: var(--radius-md); overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: var(--space-4) var(--space-2) var(--space-3); text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.pillar.liunian { background: var(--liki-surface-cream); box-shadow: inset 0 2px 0 var(--liki-primary); }
.pillar .label { font-size: var(--font-size-caption); color: var(--liki-text-3); margin-bottom: var(--space-2); letter-spacing: 0.08em; }
.pillar .gl { font-size: var(--font-size-title-lg); font-weight: 700; letter-spacing: 0.02em; }
.pillar.self .gl { color: var(--liki-primary); }
.pillar .sub { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.pillar .nayin { color: var(--liki-text-3); }
.pillar .cang { color: var(--liki-text-3); font-size: var(--font-size-micro); }
.micro-group { display: flex; flex-direction: column; gap: 2px; margin-top: var(--space-2); padding-top: var(--space-2); border-top: 1px solid var(--liki-line); }
.pillar .micro { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.pillar-tags { display: flex; flex-wrap: wrap; gap: var(--space-1); justify-content: center; margin-top: var(--space-2); }
.void-tag, .solid-tag, .line-tag { padding: 1px 8px; border-radius: var(--radius-full); font-size: var(--font-size-micro); }
.void-tag { background: var(--liki-primary-tint); color: var(--liki-primary); }
.solid-tag { background: var(--liki-primary); color: var(--liki-on-primary); }
.line-tag { border: 1px solid var(--liki-line); color: var(--liki-text-2); }
.san-yuan { display: flex; flex-wrap: wrap; gap: var(--space-2); align-items: center; margin-top: var(--space-3); font-size: var(--font-size-label); }
.sy-body { color: var(--liki-ink); }
@media (min-width: 720px) {
  .pillars.with-liunian { grid-template-columns: repeat(5, 1fr); }
}
@media (max-width: 719px) {
  .pillars.with-liunian .pillar.liunian { grid-column: 1 / -1; }
}
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
}
</style>
