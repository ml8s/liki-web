<template>
  <div
    :class="panelClasses"
    :tabindex="-1"
    :aria-labelledby="titleId"
    :aria-current="focused === 'liunian' ? 'true' : undefined"
    :data-focused="focused === 'liunian' ? 'true' : 'false'"
  >
    <div
      :id="titleId"
      class="sec-title"
    >
      {{ $t('profiles.chart.liunianDetail') }}
    </div>
    <div class="pillar liunian">
      <div class="label">{{ $t('profiles.chart.liunianPillar') }} {{ result.year }}</div>
      <div class="gl font-serif">{{ result.nian_name }}</div>
      <div
        v-if="result.shi_shen"
        class="sub"
      >{{ result.shi_shen }}</div>
      <div
        v-if="result.na_yin"
        class="sub nayin"
      >{{ result.na_yin }}</div>
      <div
        v-if="typeof result.sheng === 'number' || typeof result.ke === 'number'"
        class="micro-group"
      >
        <div class="micro">
          {{ $t('profiles.chart.shengCount') }} {{ result.sheng ?? 0 }} · {{ $t('profiles.chart.keCount') }} {{ result.ke ?? 0 }}
        </div>
      </div>
    </div>
    <div class="liunian-detail">
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
        v-if="fuyinFanyin.length"
        class="shensha ln-line"
      >
        <span class="sy-label">{{ $t('profiles.chart.fuyinFanyin') }}</span>
        <span
          v-for="(item, index) in fuyinFanyin"
          :key="index"
          class="chart-chip"
        >{{ item.detail || item.type }}</span>
      </div>
      <template
        v-for="block in interactionBlocks"
        :key="block.title"
      >
        <div class="ln-block-title">
          {{ block.title }}
        </div>
        <div class="rel-table">
          <div
            v-for="row in block.rows"
            :key="row.label"
            class="rel-row"
          >
            <span class="rel-label">{{ row.label }}</span>
            <span class="rel-chips">
              <span
                v-for="(chip, index) in row.items"
                :key="index"
                class="chart-chip"
              >
                <span class="rel-main">{{ chip.text }}</span>
                <span
                  v-if="chip.pos"
                  class="rel-pos"
                >{{ chip.pos }}</span>
              </span>
            </span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziLiuNianPanelProps } from '../contract'
import type { BaziLiuNianInteraction } from '../../../api/engine/bazi'

const props = defineProps<BaziLiuNianPanelProps>()

const { t } = useI18n()
const titleId = useId()

const panelClasses = computed(() => [
  'engine-bazi-liunian-panel',
  `engine-bazi-liunian-panel--${props.density ?? 'full'}`,
])

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

const shensha = computed(() => props.result.shensha ?? [])
const fuyinFanyin = computed(() => props.result.fuyin_fanyin ?? [])

function interactionRows(interaction?: BaziLiuNianInteraction) {
  if (!interaction) return []
  const rows: Array<{ label: string; items: Array<{ text: string; pos: string }> }> = []
  const ganItems = (interaction.gan_rels ?? [])
    .map((relation, index) => ({ text: relation.relation ?? '', pos: pillarShort(index) }))
    .filter(item => item.text)
  if (ganItems.length) rows.push({ label: t('profiles.chart.ganRel'), items: ganItems })
  const zhiItems = (interaction.zhi_rels ?? [])
    .map((relation, index) => ({ relation, pos: pillarShort(index) }))
    .filter(({ relation }) => (relation.type ?? '') !== '无' && !!relation.detail && relation.detail !== '无特殊关系')
    .map(({ relation, pos }) => ({ text: relation.detail ?? '', pos }))
  if (zhiItems.length) rows.push({ label: t('profiles.chart.zhiRel'), items: zhiItems })
  return rows
}

const interactionBlocks = computed(() => {
  const annual = props.result
  const blocks: Array<{ title: string; rows: Array<{ label: string; items: Array<{ text: string; pos: string }> }> }> = []
  const natalRows = interactionRows(annual.natal_interactions?.[0])
  if (natalRows.length) blocks.push({ title: t('profiles.chart.withNatal'), rows: natalRows })
  const dayun = annual.dayun_interactions?.[0]
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
.engine-bazi-liunian-panel { min-width: 0; }
.pillar.liunian {
  min-width: 0;
  margin-bottom: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-small);
  overflow-wrap: anywhere;
}
.pillar.liunian .label { font-size: var(--font-size-caption); color: var(--liki-text-2); }
.pillar.liunian .gl { font-size: var(--font-size-title-lg); font-weight: 700; color: var(--liki-primary); }
.pillar.liunian .sub { font-size: var(--font-size-label); color: var(--liki-text-2); }
.pillar.liunian .micro { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.engine-bazi-liunian-panel.focused,
.engine-bazi-liunian-panel[data-focused='true'] { outline: 2px solid var(--liki-primary); outline-offset: 2px; }
.liunian-detail { display: flex; flex-direction: column; gap: var(--space-2); }
.ln-line { margin-top: 0; }
.ln-block-title { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.ln-cat { margin-left: var(--space-1); font-style: normal; font-size: var(--font-size-micro); opacity: 0.85; }
</style>
