<template>
  <template v-if="rows.length">
    <div class="sec-title">{{ $t('profiles.chart.yongShen') }}</div>
    <div class="ys">
      <div
        v-for="row in rows"
        :key="row.label"
        class="ys-row"
      >
        <span class="ys-label">{{ row.label }}</span>
        <span class="ys-body">{{ row.text }}</span>
      </div>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziFullChartData } from '../../../api/engine/bazi'

const props = defineProps<{ full?: BaziFullChartData | null }>()
const { t } = useI18n()

const rows = computed(() => {
  const full = props.full
  if (!full) return []
  const { fu_yi, tiao_hou, ge_ju } = full
  const out: Array<{ label: string; text: string }> = []
  if (fu_yi?.qiangruo) {
    out.push({
      label: t('profiles.chart.fuYi'),
      text: t('profiles.chart.favorAvoid', { value: fu_yi.qiangruo, favorable: fu_yi.xi ?? '—', unfavorable: fu_yi.ji ?? '—' }),
    })
  }
  if (tiao_hou?.yong) {
    out.push({ label: t('profiles.chart.tiaoHou'), text: t('profiles.chart.useFavor', { value: tiao_hou.yong, favorable: tiao_hou.xi ?? '—' }) })
  }
  if (ge_ju?.ge_ju) {
    out.push({
      label: t('profiles.chart.geJu'),
      text: t('profiles.chart.patternValue', {
        value: `${ge_ju.ge_ju}${ge_ju.yong_fa ? ` · ${ge_ju.yong_fa}` : ''}`,
        favorable: ge_ju.xi ?? '—',
        unfavorable: ge_ju.ji ?? '—',
      }),
    })
  }
  return out
})
</script>

<style scoped src="../../../styles/paipan.css"></style>

<style scoped>
.ys { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.ys-row { display: flex; gap: var(--space-3); padding: var(--space-2) var(--space-3); font-size: var(--font-size-label); }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 auto; min-width: 3em; }
.ys-body { color: var(--liki-ink); }
</style>
