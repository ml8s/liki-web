<template>
  <div :class="panelClasses">
    <div class="pillars">
      <div
        v-for="item in pillars"
        :key="item.id"
        :class="['pillar', { self: item.id === 'ri', focused: item.id === focusedId }]"
        :tabindex="-1"
        :aria-label="`${item.label} ${item.gan}${item.zhi}`"
        :aria-current="item.id === focusedId ? 'true' : undefined"
        :data-focused="item.id === focusedId ? 'true' : 'false'"
      >
        <div class="label">
          {{ item.label }}
        </div>
        <div class="gl font-serif">
          {{ item.gan }}{{ item.zhi }}
        </div>
        <div class="sub">
          {{ item.shiShen }}
        </div>
        <div class="sub nayin">
          {{ item.naYin }}
        </div>
        <div
          v-if="item.cangGan"
          class="sub cang"
        >
          {{ item.cangGan }}
        </div>
        <div
          v-if="item.trend || item.selfSitting || item.xunKong"
          class="micro-group"
        >
          <div
            v-if="item.trend"
            class="micro"
          >
            {{ $t('profiles.chart.starLuck') }} {{ item.trend }}
          </div>
          <div
            v-if="item.selfSitting"
            class="micro"
          >
            {{ $t('profiles.chart.selfSitting') }} {{ item.selfSitting }}
          </div>
          <div
            v-if="item.xunKong"
            class="micro"
          >
            {{ $t('profiles.chart.xunKong') }} {{ item.xunKong }}
          </div>
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
        v-for="name in shenShaList"
        :key="name"
        class="chart-chip"
      >{{ name }}</span>
    </div>

    <template v-if="relationRows.length">
      <div class="sec-title">
        {{ $t('profiles.chart.relations') }}
      </div>
      <div class="rel-table">
        <div
          v-for="row in relationRows"
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
              <span class="rel-main font-serif">{{ chip.text }}</span>
              <span
                v-if="chip.element"
                class="rel-el"
              >→{{ chip.element }}</span>
              <span
                v-if="chip.pos"
                class="rel-pos"
              >{{ chip.pos }}</span>
              <span
                v-if="chip.mark"
                class="rel-mark"
              >{{ chip.mark }}</span>
            </span>
          </span>
        </div>
      </div>
    </template>
    <div
      v-if="tenGodRows.length || elementRows.length"
      class="analysis-grid"
    >
      <section
        v-if="tenGodRows.length"
        class="analysis-sec"
      >
        <div class="sec-title">
          {{ $t('profiles.chart.tenGods') }}
        </div>
        <div class="mini-table-wrap">
          <table class="mini-table">
            <thead>
              <tr>
                <th scope="col" />
                <th scope="col">
                  {{ $t('profiles.chart.marks.transparent') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.marks.hidden') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.marks.rooted') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.marks.timely') }}
                </th>
                <th
                  scope="col"
                  class="num"
                >
                  {{ $t('profiles.chart.count') }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="state in tenGodRows"
                :key="state.shi_shen"
                :class="`tg-${state.strength}`"
              >
                <th scope="row">
                  <span class="tg-name">{{ state.shi_shen }}</span><span class="tg-wx">{{ state.wuxing }}</span>
                </th>
                <td><i :class="['dot', { on: state.transparent }]" /></td>
                <td><i :class="['dot', { on: state.hidden }]" /></td>
                <td><i :class="['dot', { on: state.rooted }]" /></td>
                <td><i :class="['dot', { on: state.timely }]" /></td>
                <td class="num">
                  {{ state.count }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <section
        v-if="elementRows.length"
        class="analysis-sec"
      >
        <div class="sec-title">
          {{ $t('profiles.chart.elementStates') }}
        </div>
        <div class="mini-table-wrap">
          <table class="mini-table">
            <thead>
              <tr>
                <th scope="col" />
                <th scope="col">
                  {{ $t('profiles.chart.season') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.strength') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.marks.transparent') }}
                </th>
                <th scope="col">
                  {{ $t('profiles.chart.marks.rooted') }}
                </th>
                <th
                  scope="col"
                  class="num"
                >
                  {{ $t('profiles.chart.count') }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="state in elementRows"
                :key="state.wuxing"
                :class="`el-${state.strength}`"
              >
                <th scope="row">
                  <span class="tg-name">{{ state.wuxing }}</span>
                </th>
                <td>{{ mapped(SEASON_KEYS, state.season) }}</td>
                <td>{{ mapped(STRENGTH_KEYS, state.strength) }}</td>
                <td><i :class="['dot', { on: state.transparent }]" /></td>
                <td><i :class="['dot', { on: state.rooted }]" /></td>
                <td class="num">
                  {{ wuxingCounts[state.wuxing] ?? '' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
    <div
      v-if="luRoots.length"
      class="shensha"
    >
      <span class="sy-label">{{ $t('profiles.chart.luRoots') }}</span>
      <span
        v-for="root in luRoots"
        :key="`${root.gan}${root.zhi}`"
        class="chart-chip"
      >
        <span class="font-serif">{{ root.gan }}禄{{ root.zhi }}</span><span class="lu-shen">{{ root.shi_shen }}</span>
      </span>
    </div>

    <template v-if="daYunSteps.length">
      <div class="sec-title">
        {{ $t('profiles.chart.daYun') }}
      </div>
      <div
        v-if="daYunNote"
        class="sec-note"
      >
        {{ daYunNote }}
      </div>
      <div class="dayun">
        <div
          v-for="(step, index) in daYunSteps"
          :key="`${step.name}-${index}`"
          :class="['dy', { cur: index === currentStepIndex }]"
        >
          <div class="dy-name font-serif">
            {{ step.name }}
          </div>
          <div class="dy-shen">
            {{ step.shi_shen }}
          </div>
          <div class="dy-year">
            {{ step.start_year }}–{{ step.end_year }}
          </div>
        </div>
      </div>
    </template>
    <template v-if="changSheng.length">
      <div class="sec-title">
        {{ $t('profiles.chart.changShengStages') }}
      </div>
      <div class="cs-grid">
        <div
          v-for="stage in changSheng"
          :key="stage.index"
          :class="['cs-cell', { on: pillarZhi.set.has(stage.index), cur: stage.index === pillarZhi.month }]"
        >
          <div class="cs-zhi">
            {{ stage.index }}
          </div>
          <div class="cs-name">
            {{ stage.name }}
          </div>
        </div>
      </div>
    </template>
    <template v-if="nayinRows.length">
      <div class="sec-title">
        {{ $t('profiles.chart.nayinRel') }}
      </div>
      <div class="nayin-rel">
        <span
          v-for="(row, index) in nayinRows"
          :key="index"
          class="chart-chip"
        >
          <span class="nr-pos">{{ row.pos }}</span>
          <span class="nr-rel">{{ row.text }}</span>
        </span>
      </div>
    </template>

    <!-- Composition slot preserves the legacy position of an independently loaded annual panel. -->
    <slot name="before-yongshen" />

    <template v-if="yongShenRows.length">
      <div class="sec-title">
        {{ $t('profiles.chart.yongShen') }}
      </div>
      <div class="ys">
        <div
          v-for="row in yongShenRows"
          :key="row.label"
          class="ys-row"
        >
          <span class="ys-label">{{ row.label }}</span>
          <span class="ys-body">{{ row.text }}</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import '@shared/styles/chart-panel.css'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BaziFullChartPanelProps } from '../contract'
import type { BaziFullChartData, BaziJuRelation } from '../../../api/engine/bazi'

const props = defineProps<BaziFullChartPanelProps>()

const { t } = useI18n()

const panelClasses = computed(() => [
  'engine-bazi-full-chart-panel',
  `engine-bazi-full-chart-panel--${props.density ?? 'full'}`,
])

const focusedId = computed(() => props.focused)

const pillars = computed(() => {
  const source = props.result
  const map = (id: 'nian' | 'yue' | 'ri' | 'shi', key: 'year' | 'month' | 'day' | 'hour', pillar: BaziFullChartData['nian']) => {
    const hiddenStems = pillar.cang_gan
      ? [pillar.cang_gan.main, pillar.cang_gan.mid, pillar.cang_gan.minor].filter(Boolean).join(' ')
      : ''
    return {
      id,
      label: t(`profiles.chart.pillar.${key}`),
      self: id === 'ri',
      gan: pillar.gan,
      zhi: pillar.zhi,
      naYin: pillar.na_yin ?? '',
      shiShen: pillar.shi_shens?.[0]?.shi_shen ?? '',
      cangGan: hiddenStems ? t('profiles.chart.hiddenStems', { stems: hiddenStems }) : '',
      isVoid: !!pillar.is_void,
      trend: pillar.day_master_trend ?? '',
      selfSitting: pillar.self_sitting ?? '',
      xunKong: pillar.xun_kong ?? '',
      isKuiGang: !!pillar.is_kui_gang,
      selfHeName: pillar.self_he_name ?? '',
    }
  }

  return [
    map('nian', 'year', source.nian),
    map('yue', 'month', source.yue),
    map('ri', 'day', source.ri),
    map('shi', 'hour', source.shi),
  ]
})

function zhuText(pillar?: { gan: string; zhi: string }): string {
  return pillar ? `${pillar.gan}${pillar.zhi}` : ''
}

const sanYuanText = computed(() => {
  const sanYuan = props.result.san_yuan
  if (!sanYuan) return ''
  return t('profiles.chart.sanYuanValue', {
    taiYuan: zhuText(sanYuan.tai_yuan),
    mingGong: zhuText(sanYuan.ming_gong),
    shenGong: zhuText(sanYuan.shen_gong),
  })
})

const shenShaList = computed(() => {
  const names = new Set<string>()
  for (const pillar of [props.result.nian, props.result.yue, props.result.ri, props.result.shi]) {
    for (const shenSha of pillar.shen_sha ?? []) {
      if (shenSha.name) names.add(shenSha.name)
    }
  }
  return [...names]
})

const PILLAR_KEYS = ['year', 'month', 'day', 'hour'] as const
const ENGINE_PILLAR_INDEX: Record<string, number> = { nian: 0, yue: 1, ri: 2, shi: 3 }

function pillarShort(index: number): string {
  const key = PILLAR_KEYS[index]
  return key ? t(`profiles.chart.pillarShort.${key}`) : ''
}

function pillarIndex(key: string): number {
  return ENGINE_PILLAR_INDEX[key] ?? -1
}

function posText(...indices: number[]): string {
  return indices.map(pillarShort).filter(Boolean).join('·')
}

const SEASON_KEYS: Record<string, string> = {
  旺: 'profiles.chart.seasonValue.prosperous',
  相: 'profiles.chart.seasonValue.strong',
  休: 'profiles.chart.seasonValue.resting',
  囚: 'profiles.chart.seasonValue.trapped',
  死: 'profiles.chart.seasonValue.dead',
}
const STRENGTH_KEYS: Record<string, string> = {
  strong: 'profiles.chart.strengthValue.strong',
  neutral: 'profiles.chart.strengthValue.neutral',
  weak: 'profiles.chart.strengthValue.weak',
}

function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}

interface RelationChip {
  text: string
  element?: string
  pos?: string
  mark?: string
}

const relationRows = computed(() => {
  const full = props.result
  const rows: Array<{ label: string; items: RelationChip[] }> = []
  const add = (label: string, items: RelationChip[]) => {
    const list = items.filter(item => item.text)
    if (list.length) rows.push({ label, items: list })
  }
  const pair = (relation: { zhi_a: string; zhi_b: string; pillar_a: number; pillar_b: number }): RelationChip => ({
    text: `${relation.zhi_a}${relation.zhi_b}`,
    pos: posText(relation.pillar_a, relation.pillar_b),
  })
  const ju = (relation: BaziJuRelation): RelationChip => ({
    text: relation.name ?? '',
    pos: (relation.pillars ?? []).map(key => pillarShort(pillarIndex(key))).filter(Boolean).join('·'),
  })

  add(t('profiles.chart.rel.ganHe'), (full.gan_he ?? []).map(relation => ({
    text: `${relation.gan_a}${relation.gan_b}`,
    element: relation.he_element,
    pos: posText(relation.pillar_a, relation.pillar_b),
    mark: relation.contested ? t('profiles.chart.rel.contested') : undefined,
  })))
  add(t('profiles.chart.rel.ganChong'), (full.gan_chong ?? []).map(relation => ({
    text: `${relation.gan_a}${relation.gan_b}`,
    pos: posText(relation.pillar_a, relation.pillar_b),
  })))
  add(t('profiles.chart.rel.zhiLiuHe'), (full.zhi_liu_he ?? []).map(relation => ({
    text: `${relation.zhi_a}${relation.zhi_b}`,
    element: relation.wuxing,
    pos: posText(relation.pillar_a, relation.pillar_b),
  })))
  add(t('profiles.chart.rel.sanHe'), (full.san_he ?? []).map(ju))
  add(t('profiles.chart.rel.sanHePartial'), (full.san_he_partial ?? []).map(ju))
  add(t('profiles.chart.rel.sanHui'), (full.san_hui ?? []).map(ju))
  add(t('profiles.chart.rel.liuChong'), (full.liu_chong ?? []).map(pair))
  add(t('profiles.chart.rel.liuHai'), (full.liu_hai ?? []).map(pair))
  add(t('profiles.chart.rel.liuXing'), (full.liu_xing ?? []).map(pair))
  add(t('profiles.chart.rel.liuPo'), (full.liu_po ?? []).map(pair))
  add(t('profiles.chart.rel.anHe'), (full.an_he ?? []).map(pair))
  add(t('profiles.chart.rel.gongJia'), (full.gong_jia ?? []).map(relation => ({
    text: `${relation.type}${relation.zhi}`,
    element: relation.wuxing,
    pos: posText(relation.pillar_a, relation.pillar_b),
  })))
  return rows
})

const tenGodRows = computed(() => props.result.ten_god_states ?? [])
const elementRows = computed(() => props.result.element_states ?? [])
const wuxingCounts = computed(() => props.result.fu_yi?.wuxing_count ?? {})

const luRoots = computed(() => {
  const seen = new Set<string>()
  return (props.result.lu_roots ?? []).filter((root) => {
    const key = `${root.gan}${root.zhi}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
})

const daYunSteps = computed(() => props.result.da_yun?.steps ?? [])
const currentYear = new Date().getFullYear()
const currentStepIndex = computed(() => daYunSteps.value.findIndex(
  step => (step.start_year ?? 0) <= currentYear && currentYear <= (step.end_year ?? 0),
))
const daYunNote = computed(() => {
  const daYun = props.result.da_yun
  const parts: string[] = []
  const direction = mapped(DAYUN_DIR_KEYS, daYun?.direction)
  if (direction) parts.push(direction)
  const step = daYunSteps.value[currentStepIndex.value]
  if (step?.end_year) parts.push(t('profiles.chart.daYunNext', { years: step.end_year - currentYear }))
  return parts.join(' · ')
})
const DAYUN_DIR_KEYS: Record<string, string> = {
  顺排: 'profiles.chart.daYunDir.forward',
  逆排: 'profiles.chart.daYunDir.backward',
}
const NAYIN_KEYS: Record<string, string> = {
  相生: 'profiles.chart.nayinRelValue.sheng',
  相克: 'profiles.chart.nayinRelValue.ke',
  相同: 'profiles.chart.nayinRelValue.same',
}

const changSheng = computed(() => props.result.chang_sheng ?? [])
const pillarZhi = computed(() => ({
  set: new Set([props.result.nian.zhi, props.result.yue.zhi, props.result.ri.zhi, props.result.shi.zhi]),
  month: props.result.yue.zhi,
}))

const nayinRows = computed(() => (props.result.nayin_rel ?? []).map(relation => ({
  pos: posText(pillarIndex(relation.a), pillarIndex(relation.b)),
  text: mapped(NAYIN_KEYS, relation.relation),
})))

const yongShenRows = computed(() => {
  const full = props.result
  const { fu_yi, tiao_hou, ge_ju } = full
  const rows: Array<{ label: string; text: string }> = []
  if (fu_yi?.qiangruo) {
    rows.push({
      label: t('profiles.chart.fuYi'),
      text: t('profiles.chart.favorAvoid', { value: fu_yi.qiangruo, favorable: fu_yi.xi ?? '—', unfavorable: fu_yi.ji ?? '—' }),
    })
  }
  if (tiao_hou?.primary_wuxing) {
    const candidates = [tiao_hou.primary_wuxing, tiao_hou.secondary_wuxing]
      .filter(Boolean)
      .join(' · ')
    rows.push({ label: t('profiles.chart.tiaoHou'), text: candidates })
  }
  if (ge_ju?.ge_ju) {
    rows.push({
      label: t('profiles.chart.geJu'),
      text: `${ge_ju.ge_ju}${ge_ju.yong_fa ? ` · ${ge_ju.yong_fa}` : ''}`,
    })
  }
  return rows
})
</script>

<style scoped>
.engine-bazi-full-chart-panel { min-width: 0; overflow-wrap: anywhere; }
.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: var(--radius-md); overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: var(--space-4) var(--space-2) var(--space-3); text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
.pillar.focused { outline: 2px solid var(--liki-primary); outline-offset: -2px; }
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
.rel-el { color: var(--liki-primary); }
.rel-mark { color: var(--color-error); font-size: var(--font-size-micro); }
.analysis-grid { display: grid; gap: var(--space-2); }
.analysis-sec { min-width: 0; }
.mini-table .num { font-variant-numeric: tabular-nums; color: var(--liki-text-2); }
.tg-wx { margin-left: var(--space-1); font-size: var(--font-size-micro); color: var(--liki-text-3); font-weight: 400; }
.dot { display: inline-block; width: 7px; height: 7px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); }
.dot.on { background: var(--liki-primary); border-color: var(--liki-primary); }
.tg-strong .tg-name, .el-strong .tg-name { color: var(--liki-primary); }
.tg-weak .tg-name, .el-weak .tg-name { color: var(--liki-text-3); }
.lu-shen { margin-left: var(--space-1); color: var(--liki-text-3); font-size: var(--font-size-micro); }
.sec-note { font-size: var(--font-size-caption); color: var(--liki-text-3); margin: calc(-1 * var(--space-1)) 0 var(--space-2); }
.dy.cur .dy-name, .dy.cur .dy-shen { color: var(--liki-primary); }
.cs-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(48px, 1fr)); gap: var(--space-2); }
.cs-cell { border: 1px solid var(--liki-line); border-radius: var(--radius-small); padding: 6px 2px; text-align: center; }
.cs-cell.on { background: var(--liki-surface-cream); }
.cs-cell.cur { background: var(--liki-primary-tint); border-color: var(--liki-primary-border); }
.cs-zhi { font-size: var(--font-size-caption); font-weight: 600; }
.cs-cell.cur .cs-zhi { color: var(--liki-primary); }
.cs-name { font-size: var(--font-size-micro); color: var(--liki-text-3); margin-top: 2px; }
.nayin-rel { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.nr-pos { color: var(--liki-text-3); }
.nr-rel { color: var(--liki-ink); }
.ys { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.ys-row { display: flex; gap: var(--space-3); padding: var(--space-2) var(--space-3); font-size: var(--font-size-label); }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 auto; min-width: 3em; }
.ys-body { color: var(--liki-ink); }

@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
}
@media (min-width: 640px) {
  .analysis-grid { grid-template-columns: 1fr 1fr; gap: var(--space-4); }
}
</style>
