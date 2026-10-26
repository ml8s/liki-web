<template>
  <div class="liuyao-panel liuyao-chart-panel">
    <h2 class="sec-title">
      {{ $t('services.divination.hexagram') }}
    </h2>
    <div class="gua-row">
      <div class="gua-name font-serif">
        {{ result.name }}
      </div>
      <div class="gua-sub">
        <span v-if="result.ben_gua">{{ $t('services.divination.original') }} {{ result.ben_gua }}</span>
        <span v-if="result.bian_gua"> · {{ $t('services.divination.changed') }} {{ result.bian_gua }}</span>
      </div>
    </div>
    <div class="gua-info">
      <span v-if="result.gong">{{ $t('services.divination.palace', { palace: result.gong }) }}</span>
      <span v-if="result.gong_wuxing"> · {{ result.gong_wuxing }}</span>
      <span v-if="result.ri_chen_gan"> · {{ $t('services.divination.dayPillar') }} {{ result.ri_chen_gan }}{{ result.ri_chen_zhi }}</span>
      <span v-if="result.yue_jian_zhi"> · {{ $t('services.divination.monthBranch') }} {{ result.yue_jian_gan }}{{ result.yue_jian_zhi }}</span>
      <span v-if="xunKongText"> · {{ $t('profiles.chart.void') }} {{ xunKongText }}</span>
    </div>

    <div
      v-if="lines.length"
      class="lines-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.lines') }}
      </div>
      <div class="lines-table-wrap">
        <div class="lines-table">
          <div class="line-row line-header">
            <span class="lc">{{ $t('services.divination.position') }}</span>
            <span class="lc">{{ $t('services.divination.relation') }}</span>
            <span class="lc">{{ $t('services.divination.branch') }}</span>
            <span class="lc">{{ $t('services.divination.element') }}</span>
            <span class="lc">{{ $t('services.divination.stem') }}</span>
            <span class="lc">{{ $t('services.divination.strength') }}</span>
            <span class="lc">{{ $t('services.divination.dayRelation') }}</span>
          </div>
          <div
            v-for="(line, idx) in lines"
            :key="idx"
            :class="['line-row', { dong: isDong(idx) }]"
          >
            <span class="lc">{{ $t('services.divination.lineNumber', { number: 6 - idx }) }}</span>
            <span class="lc">{{ line.liu_qin ?? '—' }}</span>
            <span class="lc">{{ line.zhi ?? '—' }}</span>
            <span class="lc">{{ line.wuxing ?? '—' }}</span>
            <span class="lc">{{ line.gan ?? '—' }}</span>
            <span class="lc">{{ wangShuai[idx] ?? '—' }}</span>
            <span class="lc">{{ dayRelations[idx]?.relations?.join('、') ?? '—' }}</span>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="bianLines.length"
      class="lines-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.changingLines') }}
      </div>
      <div class="lines-table">
        <div
          v-for="(line, idx) in bianLines"
          :key="idx"
          class="line-row"
        >
          <span class="lc">{{ $t('services.divination.lineNumber', { number: 6 - idx }) }}</span>
          <span class="lc">{{ line.liu_qin ?? '—' }}</span>
          <span class="lc">{{ line.zhi ?? '—' }}</span>
          <span class="lc">{{ line.gan ?? '—' }}</span>
        </div>
      </div>
    </div>

    <div
      v-if="focusText"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.usefulGod') }}
      </div>
      <div class="rel-text">
        {{ focusText }}
      </div>
      <div
        v-if="yongShenDetail"
        class="rel-sub"
      >
        {{ yongShenDetail }}
      </div>
    </div>

    <div
      v-if="guaCi"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.hexagramText') }}
      </div>
      <div class="rel-text">
        {{ guaCi }}
      </div>
    </div>

    <div
      v-if="patterns.length"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.patterns') }}
      </div>
      <div
        v-for="p in patterns"
        :key="`${p.type}-${p.sub_type}-${p.position}`"
        class="pattern-item"
      >
        <span class="pattern-name">{{ p.sub_type }}</span>
        <span class="pattern-desc">{{ p.type }} · 爻位 {{ p.position }} · {{ p.is_true ? '有效' : '候选' }}</span>
      </div>
    </div>

    <div
      v-if="timingText"
      class="rel-sec"
    >
      <div class="sec-title">
        {{ $t('services.divination.timing') }}
      </div>
      <div class="rel-text">
        {{ timingText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
// Provenance: extracted from web/app/src/views/DivinationView.vue.
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LiuyaoChartPanelProps } from '../contract'

const props = defineProps<LiuyaoChartPanelProps>()
const { t } = useI18n()

const lines = computed(() => props.result?.lines ?? [])
const bianLines = computed(() => props.result?.bian_yao ?? [])
const wangShuai = computed(() => props.result?.wang_shuai ?? [])
const dayRelations = computed(() => props.result?.ri_chen_relations ?? [])
const dongYao = computed(() => props.result?.casting?.dong_yao ?? [])
const patterns = computed(() => props.result?.patterns ?? [])

function isDong(idx: number) {
  const pos = 6 - idx
  return dongYao.value.includes(pos as 1 | 2 | 3 | 4 | 5 | 6)
}

const xunKongText = computed(() => {
  const xk = props.result?.xun_kong
  if (!xk || !xk.length) return ''
  return xk.join(' ')
})

const focusText = computed(() => {
  const ys = props.result?.yong_shen
  if (!ys?.name) return ''
  const position = ys.position && ys.position > 0
    ? t('services.divination.positionValue', { value: ys.position })
    : t('services.divination.hidden')
  return `${ys.name} · ${position}`
})

const yongShenDetail = computed(() => {
  const ys = props.result?.yong_shen
  if (!ys) return ''
  const parts: string[] = []
  if (ys.name) parts.push(ys.name)
  if (ys.position) parts.push(t('services.divination.positionValue', { value: ys.position }))
  return parts.length ? t('services.divination.usefulGodDetail', { value: parts.join(t('common.listSeparator')) }) : ''
})

const guaCi = computed(() => {
  const gc = props.result?.gua_ci
  if (!gc) return ''
  return [gc.name, gc.gua_ci].filter(Boolean).join('：')
})

const timingText = computed(() => {
  const tc = props.result?.timing_candidates
  if (!tc || !tc.length) return ''
  return tc.map(item => [item.window || item.id, item.mechanism, item.trigger_branch].filter(Boolean).join('：')).join('；')
})
</script>

<style scoped>
.gua-row { display: flex; align-items: baseline; gap: var(--space-3); }
.gua-name { font-size: var(--font-size-display-sm); font-weight: 700; }
.gua-sub { color: var(--liki-text-2); font-size: var(--font-size-label); }
.gua-info { margin-top: var(--space-2); font-size: var(--font-size-label); color: var(--liki-text-2); }
/* 区块间距由 chart-panel.css 的 .sec-title 提供，保持共享面板与页面主题一致。 */
.lines-table-wrap { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.lines-table { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; font-size: var(--font-size-caption); min-width: 480px; }
.line-row { display: flex; }
.line-row + .line-row { border-top: 1px solid var(--liki-line); }
.line-header { background: var(--liki-surface); font-weight: 600; }
.line-row .lc { flex: 1; padding: 6px 4px; text-align: center; }
.line-row.dong { background: var(--liki-primary-tint); }
.rel-text { font-size: var(--font-size-body); color: var(--liki-ink); }
.rel-sub { font-size: var(--font-size-label); color: var(--liki-text-2); margin-top: var(--space-1); }
.pattern-item { display: flex; flex-direction: column; gap: 2px; margin-bottom: var(--space-1); }
.pattern-name { font-weight: 600; color: var(--liki-primary); }
.pattern-desc { font-size: var(--font-size-caption); color: var(--liki-text-3); }
</style>
