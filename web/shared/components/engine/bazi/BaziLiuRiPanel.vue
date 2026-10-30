<template>
  <section
    :class="panelClasses"
    :tabindex="-1"
    :aria-labelledby="titleId"
  >
    <div
      :id="titleId"
      class="sec-title"
    >
      {{ $t('profiles.chart.liuri') }}
    </div>

    <div class="pillar liunian">
      <div class="label">{{ result.date }}</div>
      <div class="gl font-serif">{{ result.day_name }}</div>
      <div
        v-if="result.shi_shen"
        class="sub"
      >{{ result.shi_shen }}</div>
      <div
        v-if="result.day_nayin"
        class="sub nayin"
      >{{ result.day_nayin }}</div>
    </div>

    <div
      v-if="result.resolved?.local_date"
      class="liuri-resolved"
    >
      {{ $t('profiles.chart.liuriLocalDate') }} {{ result.resolved.local_date }}
    </div>

    <div class="liuri-detail">
      <div
        v-if="shensha.length"
        class="shensha ln-line"
      >
        <span class="sy-label">{{ $t('profiles.chart.shenSha') }}</span>
        <span
          v-for="item in shensha"
          :key="item.name"
          :class="['chart-chip', categoryClass(item.category)]"
          :title="item.description"
        >{{ item.name }}<i class="ln-cat">{{ categoryLabel(item.category) }}</i></span>
      </div>

      <div
        v-for="block in relationBlocks"
        :key="block.title"
        class="liuri-block"
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
                v-for="(chip, index) in row.chips"
                :key="index"
                class="chart-chip"
              >{{ chip }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziLiuRiPanelProps } from '../contract'
import type {
  BaziLiuRiGanRel,
  BaziLiuRiZhiRel,
} from '../../../api/engine/bazi'

const props = defineProps<BaziLiuRiPanelProps>()
const { t } = useI18n()
const titleId = useId()

const panelClasses = computed(() => [
  'engine-bazi-liuri-panel',
  `engine-bazi-liuri-panel--${props.density ?? 'full'}`,
])

const shensha = computed(() => props.result.shensha ?? [])

function categoryClass(category?: string): string {
  if (category === '吉') return 'good'
  if (category === '凶') return 'bad'
  return ''
}

function categoryLabel(category?: string): string {
  if (category === '吉') return t('profiles.chart.auspicious')
  if (category === '凶') return t('profiles.chart.inauspicious')
  return ''
}

function ganChips(items?: BaziLiuRiGanRel[]): string[] {
  return (items ?? [])
    .map(item => item.relation)
    .filter((value): value is string => !!value)
}

function zhiChips(items?: BaziLiuRiZhiRel[]): string[] {
  return (items ?? [])
    .filter(item => item.detail && item.detail !== '无特殊关系')
    .map(item => item.detail!)
}

const relationBlocks = computed(() => {
  const blocks: Array<{ title: string; rows: Array<{ label: string; chips: string[] }> }> = []
  const natalGan = ganChips(props.result.gan_rels)
  const natalZhi = zhiChips(props.result.zhi_rels)
  if (natalGan.length || natalZhi.length) {
    blocks.push({
      title: t('profiles.chart.withNatal'),
      rows: [
        { label: t('profiles.chart.ganRel'), chips: natalGan },
        { label: t('profiles.chart.zhiRel'), chips: natalZhi },
      ].filter(row => row.chips.length),
    })
  }
  const dayun = zhiChips(props.result.dayun_rels)
  if (dayun.length) {
    blocks.push({ title: t('profiles.chart.withDaYun'), rows: [{ label: t('profiles.chart.zhiRel'), chips: dayun }] })
  }
  const annual = zhiChips(props.result.liunian_rels)
  if (annual.length) {
    blocks.push({ title: t('profiles.chart.withAnnual'), rows: [{ label: t('profiles.chart.zhiRel'), chips: annual }] })
  }
  return blocks.filter(block => block.rows.length)
})
</script>

<style scoped>
.liuri-resolved {
  margin-top: var(--space-1);
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
  font-variant-numeric: tabular-nums;
}

.liuri-detail {
  margin-top: var(--space-3);
  display: grid;
  gap: var(--space-3);
}

.liuri-block .ln-block-title {
  margin-bottom: var(--space-2);
}
</style>
