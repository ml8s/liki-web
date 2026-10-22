<template>
  <div
    v-if="chart"
    class="bazi-panel"
  >
    <div :class="['pillars', { 'with-liunian': !!liunian }]">
      <div
        v-for="item in baziPillars"
        :key="item.label"
        :class="['pillar', { self: item.self }]"
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
      <div
        v-if="liunian"
        class="pillar liunian"
      >
        <div class="label">
          {{ $t('profiles.chart.liunianPillar') }} {{ liunian.year }}
        </div>
        <div class="gl font-serif">
          {{ liunian.nian_name }}
        </div>
        <div
          v-if="liunian.shi_shen"
          class="sub"
        >
          {{ liunian.shi_shen }}
        </div>
        <div
          v-if="liunian.na_yin"
          class="sub nayin"
        >
          {{ liunian.na_yin }}
        </div>
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
        class="paipan-chip"
      >{{ s }}</span>
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
              v-for="(c, i) in row.items"
              :key="i"
              class="paipan-chip"
            >
              <span class="rel-main font-serif">{{ c.text }}</span>
              <span
                v-if="c.el"
                class="rel-el"
              >→{{ c.el }}</span>
              <span
                v-if="c.pos"
                class="rel-pos"
              >{{ c.pos }}</span>
              <span
                v-if="c.mark"
                class="rel-mark"
              >{{ c.mark }}</span>
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
                v-for="s in tenGodRows"
                :key="s.shi_shen"
                :class="`tg-${s.strength}`"
              >
                <th scope="row">
                  <span class="tg-name">{{ s.shi_shen }}</span><span class="tg-wx">{{ s.wuxing }}</span>
                </th>
                <td><i :class="['dot', { on: s.transparent }]" /></td>
                <td><i :class="['dot', { on: s.hidden }]" /></td>
                <td><i :class="['dot', { on: s.rooted }]" /></td>
                <td><i :class="['dot', { on: s.timely }]" /></td>
                <td class="num">
                  {{ s.count }}
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
                v-for="e in elementRows"
                :key="e.wuxing"
                :class="`el-${e.strength}`"
              >
                <th scope="row">
                  <span class="tg-name">{{ e.wuxing }}</span>
                </th>
                <td>{{ mapped(SEASON_KEYS, e.season) }}</td>
                <td>{{ mapped(STRENGTH_KEYS, e.strength) }}</td>
                <td><i :class="['dot', { on: e.transparent }]" /></td>
                <td><i :class="['dot', { on: e.rooted }]" /></td>
                <td class="num">
                  {{ wuxingCounts[e.wuxing] ?? '' }}
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
        v-for="r in luRoots"
        :key="`${r.gan}${r.zhi}`"
        class="paipan-chip"
      >
        <span class="font-serif">{{ r.gan }}禄{{ r.zhi }}</span><span class="lu-shen">{{ r.shi_shen }}</span>
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
          v-for="(s, i) in daYunSteps"
          :key="`${s.name}-${i}`"
          :class="['dy', { cur: i === currentStepIndex }]"
        >
          <div class="dy-name font-serif">
            {{ s.name }}
          </div>
          <div class="dy-shen">
            {{ s.shi_shen }}
          </div>
          <div class="dy-year">
            {{ s.start_year }}–{{ s.end_year }}
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
          v-for="c in changSheng"
          :key="c.index"
          :class="['cs-cell', { on: pillarZhi.set.has(c.index), cur: c.index === pillarZhi.month }]"
        >
          <div class="cs-zhi">
            {{ c.index }}
          </div>
          <div class="cs-name">
            {{ c.name }}
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
          v-for="(r, i) in nayinRows"
          :key="i"
          class="paipan-chip"
        >
          <span class="nr-pos">{{ r.pos }}</span>
          <span class="nr-rel">{{ r.text }}</span>
        </span>
      </div>
    </template>
    <template v-if="liunian">
      <div class="sec-title">
        {{ $t('profiles.chart.liunianDetail') }}
      </div>
      <div class="liunian-detail">
        <div
          v-if="liunianShensha.length"
          class="shensha ln-line"
        >
          <span class="sy-label">{{ $t('profiles.chart.shenSha') }}</span>
          <span
            v-for="s in liunianShensha"
            :key="s.name"
            :class="['paipan-chip', categoryClass(s.category)]"
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
            class="paipan-chip"
          >{{ f.detail || f.type }}</span>
        </div>
        <template
          v-for="block in liunianInteractionBlocks"
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
                  v-for="(c, i) in row.items"
                  :key="i"
                  class="paipan-chip"
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
    <template v-if="yongShenSummary.length">
      <div class="sec-title">
        {{ $t('profiles.chart.yongShen') }}
      </div>
      <div class="ys">
        <div
          v-for="row in yongShenSummary"
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
// 八字排盘展示（四柱/藏干/神煞/关系/十神力量/五行状态/大运/长生/纳音/用神）——纯展示、数据驱动。
// App 的 PaipanPreview 与营销站 island 共用；文案由宿主安装的 vue-i18n 提供，
// 键沿用 App 的 profiles.chart.*（共享拷贝见 web/shared/i18n/chart-messages.<locale>.json）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  BaziChartData,
  BaziFullChartData,
  BaziJuRelation,
  BaziLiuNianInteraction,
  BaziLiuNianOutput,
  BaziZhuFull,
} from '../../api/engine'

const props = defineProps<{
  chart: BaziChartData | null
  full?: BaziFullChartData | null
  /** 流年盘（bazi.liunian 结果）；传入时四柱旁追加流年列并展示流年详览。 */
  liunian?: BaziLiuNianOutput | null
}>()

const { t } = useI18n()

const PILLAR_KEYS = ['year', 'month', 'day', 'hour'] as const

// 引擎 wire 柱位键（合局 pillars / 纳音关系 a,b）→ 柱位下标。
const ENGINE_PILLAR_INDEX: Record<string, number> = { nian: 0, yue: 1, ri: 2, shi: 3 }

function pillarIndex(key: string): number {
  return ENGINE_PILLAR_INDEX[key] ?? -1
}

function pillarShort(index: number): string {
  const key = PILLAR_KEYS[index]
  return key ? t(`profiles.chart.pillarShort.${key}`) : ''
}

function posText(...indices: number[]): string {
  return indices.map(pillarShort).filter(Boolean).join('·')
}

// 引擎值 → i18n 键映射；未知值原样透传（传统术语保持中文）。
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
const NAYIN_KEYS: Record<string, string> = {
  相生: 'profiles.chart.nayinRelValue.sheng',
  相克: 'profiles.chart.nayinRelValue.ke',
  相同: 'profiles.chart.nayinRelValue.same',
}
const DAYUN_DIR_KEYS: Record<string, string> = {
  顺排: 'profiles.chart.daYunDir.forward',
  逆排: 'profiles.chart.daYunDir.backward',
}

function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}

const baziPillars = computed(() => {
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
  return [
    map('year', src.nian),
    map('month', src.yue),
    map('day', src.ri),
    map('hour', src.shi),
  ]
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

interface RelationChip {
  text: string
  el?: string
  pos?: string
  mark?: string
}

const relationRows = computed(() => {
  const f = props.full
  if (!f) return []
  const rows: Array<{ label: string; items: RelationChip[] }> = []
  const add = (label: string, items: RelationChip[]) => {
    const list = items.filter(i => i.text)
    if (list.length) rows.push({ label, items: list })
  }
  const pair = (r: { zhi_a: string; zhi_b: string; pillar_a: number; pillar_b: number }): RelationChip => ({
    text: `${r.zhi_a}${r.zhi_b}`,
    pos: posText(r.pillar_a, r.pillar_b),
  })
  const ju = (r: BaziJuRelation): RelationChip => ({
    text: r.name ?? '',
    pos: (r.pillars ?? []).map(p => pillarShort(pillarIndex(p))).filter(Boolean).join('·'),
  })

  add(t('profiles.chart.rel.ganHe'), (f.gan_he ?? []).map(r => ({
    text: `${r.gan_a}${r.gan_b}`,
    el: r.he_element,
    pos: posText(r.pillar_a, r.pillar_b),
    mark: r.contested ? t('profiles.chart.rel.contested') : undefined,
  })))
  add(t('profiles.chart.rel.ganChong'), (f.gan_chong ?? []).map(r => ({
    text: `${r.gan_a}${r.gan_b}`,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  add(t('profiles.chart.rel.zhiLiuHe'), (f.zhi_liu_he ?? []).map(r => ({
    text: `${r.zhi_a}${r.zhi_b}`,
    el: r.wuxing,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  add(t('profiles.chart.rel.sanHe'), (f.san_he ?? []).map(ju))
  add(t('profiles.chart.rel.sanHePartial'), (f.san_he_partial ?? []).map(ju))
  add(t('profiles.chart.rel.sanHui'), (f.san_hui ?? []).map(ju))
  add(t('profiles.chart.rel.liuChong'), (f.liu_chong ?? []).map(pair))
  add(t('profiles.chart.rel.liuHai'), (f.liu_hai ?? []).map(pair))
  add(t('profiles.chart.rel.liuXing'), (f.liu_xing ?? []).map(pair))
  add(t('profiles.chart.rel.liuPo'), (f.liu_po ?? []).map(pair))
  add(t('profiles.chart.rel.anHe'), (f.an_he ?? []).map(pair))
  add(t('profiles.chart.rel.gongJia'), (f.gong_jia ?? []).map(r => ({
    text: `${r.type}${r.zhi}`,
    el: r.wuxing,
    pos: posText(r.pillar_a, r.pillar_b),
  })))
  return rows
})

const tenGodRows = computed(() => props.full?.ten_god_states ?? [])
const elementRows = computed(() => props.full?.element_states ?? [])
const wuxingCounts = computed(() => props.full?.fu_yi?.wuxing_count ?? {})

const luRoots = computed(() => {
  const seen = new Set<string>()
  return (props.full?.lu_roots ?? []).filter(r => {
    const key = `${r.gan}${r.zhi}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
})

const changSheng = computed(() => props.full?.chang_sheng ?? [])
const pillarZhi = computed(() => {
  const src = props.full ?? props.chart
  if (!src) return { set: new Set<string>(), month: '' }
  return {
    set: new Set([src.nian.zhi, src.yue.zhi, src.ri.zhi, src.shi.zhi]),
    month: src.yue.zhi,
  }
})

const nayinRows = computed(() => (props.full?.nayin_rel ?? []).map(r => ({
  pos: posText(pillarIndex(r.a), pillarIndex(r.b)),
  text: mapped(NAYIN_KEYS, r.relation),
})))

// ── 流年盘（bazi.liunian）──
const liunianShensha = computed(() => props.liunian?.shensha ?? [])
const liunianFuyin = computed(() => props.liunian?.fuyin_fanyin ?? [])

const CATEGORY_KEYS: Record<string, string> = {
  吉: 'profiles.chart.auspicious',
  凶: 'profiles.chart.inauspicious',
}

function categoryClass(category?: string): string {
  if (category === '吉') return 'good'
  if (category === '凶') return 'bad'
  return ''
}

function categoryLabel(category?: string): string {
  return mapped(CATEGORY_KEYS, category)
}

// 行运关系：一条柱（流年/大运）对命局四柱的干支关系（gan_rels/zhi_rels 按柱位顺序）。
function interactionRows(inter?: BaziLiuNianInteraction) {
  if (!inter) return []
  const rows: Array<{ label: string; items: RelationChip[] }> = []
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
  const blocks: Array<{ title: string; rows: Array<{ label: string; items: RelationChip[] }> }> = []
  const natalRows = interactionRows(ln.natal_interactions?.[0])
  if (natalRows.length) blocks.push({ title: t('profiles.chart.withNatal'), rows: natalRows })
  const dayun = ln.dayun_interactions?.[0]
  const dayunRows = interactionRows(dayun)
  if (dayunRows.length) {
    blocks.push({
      title: dayun?.pillar_label
        ? `${t('profiles.chart.withDaYun')} · ${dayun.pillar_label}`
        : t('profiles.chart.withDaYun'),
      rows: dayunRows,
    })
  }
  return blocks
})

const daYunSteps = computed(() => props.chart?.da_yun?.steps ?? [])
const currentYear = new Date().getFullYear()
const currentStepIndex = computed(() => daYunSteps.value.findIndex(
  (step) => (step.start_year ?? 0) <= currentYear && currentYear <= (step.end_year ?? 0),
))
const daYunNote = computed(() => {
  const dy = props.chart?.da_yun ?? props.full?.da_yun
  const parts: string[] = []
  const dir = mapped(DAYUN_DIR_KEYS, dy?.direction)
  if (dir) parts.push(dir)
  const step = daYunSteps.value[currentStepIndex.value]
  if (step?.end_year) {
    parts.push(t('profiles.chart.daYunNext', { years: step.end_year - currentYear }))
  }
  return parts.join(' · ')
})

const yongShenSummary = computed(() => {
  const full = props.full
  if (!full) return []
  const { fu_yi, tiao_hou, ge_ju } = full
  const rows: Array<{ label: string; text: string }> = []

  if (fu_yi?.qiangruo) {
    rows.push({
      label: t('profiles.chart.fuYi'),
      text: t('profiles.chart.favorAvoid', { value: fu_yi.qiangruo, favorable: fu_yi.xi ?? '—', unfavorable: fu_yi.ji ?? '—' }),
    })
  }
  if (tiao_hou?.yong) {
    rows.push({ label: t('profiles.chart.tiaoHou'), text: t('profiles.chart.useFavor', { value: tiao_hou.yong, favorable: tiao_hou.xi ?? '—' }) })
  }
  if (ge_ju?.ge_ju) {
    rows.push({
      label: t('profiles.chart.geJu'),
      text: t('profiles.chart.patternValue', {
        value: `${ge_ju.ge_ju}${ge_ju.yong_fa ? ` · ${ge_ju.yong_fa}` : ''}`,
        favorable: ge_ju.xi ?? '—',
        unfavorable: ge_ju.ji ?? '—',
      }),
    })
  }
  return rows
})
</script>

<style scoped src="../../styles/paipan.css"></style>

<style scoped>
.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: var(--radius-md); overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: var(--space-4) var(--space-2) var(--space-3); text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-primary-tint), var(--liki-surface-cream)); }
/* 流年列：独立于命局四柱（引擎 bazi.liunian），顶部主色标出。 */
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

.san-yuan, .shensha { display: flex; flex-wrap: wrap; gap: var(--space-2); align-items: center; margin-top: var(--space-3); font-size: var(--font-size-label); }
.sy-label { color: var(--liki-text-3); flex: 0 0 auto; min-width: 2em; }
.sy-body { color: var(--liki-ink); }
.shensha { gap: var(--space-2); }
.lu-shen { margin-left: var(--space-1); color: var(--liki-text-3); font-size: var(--font-size-micro); }

.sec-note { font-size: var(--font-size-caption); color: var(--liki-text-3); margin: calc(-1 * var(--space-1)) 0 var(--space-2); }

.rel-table { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.rel-row { display: grid; grid-template-columns: minmax(5.5em, max-content) 1fr; gap: var(--space-2); padding: var(--space-2) var(--space-3); font-size: var(--font-size-label); align-items: baseline; }
.rel-row + .rel-row { border-top: 1px solid var(--liki-line); }
.rel-label { color: var(--liki-text-3); }
.rel-chips { display: flex; flex-wrap: wrap; gap: var(--space-2); }
.rel-main { color: var(--liki-ink); font-weight: 600; }
.rel-el { color: var(--liki-primary); }
.rel-pos { color: var(--liki-text-3); font-size: var(--font-size-micro); }
.rel-mark { color: var(--color-error); font-size: var(--font-size-micro); }

.analysis-grid { display: grid; gap: var(--space-2); }
.analysis-sec { min-width: 0; }
.mini-table .num { font-variant-numeric: tabular-nums; color: var(--liki-text-2); }
.tg-wx { margin-left: var(--space-1); font-size: var(--font-size-micro); color: var(--liki-text-3); font-weight: 400; }
.dot { display: inline-block; width: 7px; height: 7px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); }
.dot.on { background: var(--liki-primary); border-color: var(--liki-primary); }
.tg-strong .tg-name, .el-strong .tg-name { color: var(--liki-primary); }
.tg-weak .tg-name, .el-weak .tg-name { color: var(--liki-text-3); }

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

.liunian-detail { display: flex; flex-direction: column; gap: var(--space-2); }
.ln-line { margin-top: 0; }
.ln-block-title { font-size: var(--font-size-caption); color: var(--liki-text-2); margin-top: var(--space-1); }
.ln-cat { margin-left: var(--space-1); font-style: normal; font-size: var(--font-size-micro); opacity: 0.85; }

.ys { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.ys-row { display: flex; gap: var(--space-3); padding: var(--space-2) var(--space-3); font-size: var(--font-size-label); }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 auto; min-width: 3em; }
.ys-body { color: var(--liki-ink); }

@media (min-width: 640px) {
  .analysis-grid { grid-template-columns: 1fr 1fr; gap: var(--space-4); }
}
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
