<template>
  <template v-if="liunian">
    <div class="sec-title">{{ $t('profiles.chart.liunianDetail') }}</div>
    <div class="liunian-detail">
      <div
        v-if="liunianShensha.length"
        class="shensha ln-line"
      >
        <span class="sy-label">{{ $t('profiles.chart.shenSha') }}</span>
        <span
          v-for="s in liunianShensha"
          :key="s.name"
          :class="['chart-chip', categoryClass(s.category)]"
          :title="s.description"
        >{{ s.name }}<i class="ln-cat">{{ categoryLabel(s.category) }}</i></span>
      </div>
      <div
        v-if="liunianFuyin.length"
        class="shensha ln-line"
      >
        <span class="sy-label">{{ $t('profiles.chart.fuyinFanyin') }}</span>
        <span
          v-for="(f, i) in liunianFuyin"
          :key="i"
          class="chart-chip"
        >{{ f.detail || f.type }}</span>
      </div>
      <template
        v-for="block in liunianInteractionBlocks"
        :key="block.title"
      >
        <div class="ln-block-title">{{ block.title }}</div>
        <div class="rel-table">
          <div
            v-for="row in block.rows"
            :key="row.label"
            class="rel-row"
          >
            <span class="rel-label">{{ row.label }}</span>
            <span class="rel-chips">
              <span
                v-for="(c, i) in row.items"
                :key="i"
                class="chart-chip"
              >
                <span class="rel-main">{{ c.text }}</span>
                <span
                  v-if="c.pos"
                  class="rel-pos"
                >{{ c.pos }}</span>
              </span>
            </span>
          </div>
        </div>
      </template>
    </div>
  </template>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziChartData, BaziFullChartData, BaziLiuNianInteraction, BaziLiuNianOutput } from '../../../api/engine/bazi'

const props = defineProps<{
  chart: BaziChartData | null
  full?: BaziFullChartData | null
  liunian?: BaziLiuNianOutput | null
}>()

const { t } = useI18n()
const PILLAR_KEYS = ['year', 'month', 'day', 'hour'] as const
const CATEGORY_KEYS: Record<string, string> = {
  吉: 'profiles.chart.auspicious',
  凶: 'profiles.chart.inauspicious',
}

function pillarShort(index: number): string {
  const key = PILLAR_KEYS[index]
  return key ? t(`profiles.chart.pillarShort.${key}`) : ''
}
function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}
function categoryClass(category?: string): string {
  if (category === '吉') return 'good'
  if (category === '凶') return 'bad'
  return ''
}
function categoryLabel(category?: string): string {
  return mapped(CATEGORY_KEYS, category)
}

const liunianShensha = computed(() => props.liunian?.shensha ?? [])
const liunianFuyin = computed(() => props.liunian?.fuyin_fanyin ?? [])

function interactionRows(inter?: BaziLiuNianInteraction) {
  if (!inter) return []
  const rows: Array<{ label: string; items: Array<{ text: string; pos: string }> }> = []
  const ganItems = (inter.gan_rels ?? [])
    .map((r, i) => ({ text: r.relation ?? '', pos: pillarShort(i) }))
    .filter(item => item.text)
  if (ganItems.length) rows.push({ label: t('profiles.chart.ganRel'), items: ganItems })
  const zhiItems = (inter.zhi_rels ?? [])
    .map((r, i) => ({ r, pos: pillarShort(i) }))
    .filter(({ r }) => (r.type ?? '') !== '无' && !!r.detail && r.detail !== '无特殊关系')
    .map(({ r, pos }) => ({ text: r.detail ?? '', pos }))
  if (zhiItems.length) rows.push({ label: t('profiles.chart.zhiRel'), items: zhiItems })
  return rows
}

const liunianInteractionBlocks = computed(() => {
  const ln = props.liunian
  if (!ln) return []
  const blocks: Array<{ title: string; rows: Array<{ label: string; items: Array<{ text: string; pos: string }> }> }> = []
  const natalRows = interactionRows(ln.natal_interactions?.[0])
  if (natalRows.length) blocks.push({ title: t('profiles.chart.withNatal'), rows: natalRows })
  const dayun = ln.dayun_interactions?.[0]
  const dayunRows = interactionRows(dayun)
  if (dayunRows.length) {
    blocks.push({
      title: dayun?.pillar_label ? `${t('profiles.chart.withDaYun')} · ${dayun.pillar_label}` : t('profiles.chart.withDaYun'),
      rows: dayunRows,
    })
  }
  return blocks
})
</script>


<style scoped>
.liunian-detail { display: flex; flex-direction: column; gap: var(--space-2); }
.ln-line { margin-top: 0; }
.ln-block-title { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.ln-cat { margin-left: var(--space-1); font-style: normal; font-size: var(--font-size-micro); opacity: 0.85; }
</style>
