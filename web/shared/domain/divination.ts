import {
  engineLiuyaoChart,
  engineLiuyaoQigua,
  type LiuyaoChartResult,
  type LiuyaoCasting,
  type LiuyaoQiguaInput,
} from '../api/engine/liuyao'
import type { ChartInput } from './time'

/** Generate a casting receipt; chart interpretation is a separate engine call. */
export type { LiuyaoCasting, LiuyaoChartResult, LiuyaoQiguaInput }

export function divinationCastingFor(input: LiuyaoQiguaInput) {
  return engineLiuyaoQigua(input)
}

export interface DivinationChartInput extends ChartInput {
  casting: LiuyaoCasting
  yongShen?: string
}

/** Explicit casting time + location + receipt -> complete divination chart. */
export function divinationChartFor(
  input: DivinationChartInput,
): Promise<LiuyaoChartResult> {
  return engineLiuyaoChart(input)
}

// 六爻起卦（免费层）的本卦/变卦推断：由 casting 的六爻值自行推演，无需求解卦。
// 上下卦名表由 engine 的 liuyao/data/hexagrams.json 生成（同一数据源，勿手改）。

export interface YiGuaReading {
  /** 自下而上的六爻值（6 老阴 · 7 少阳 · 8 少阴 · 9 老阳）。 */
  yaos: number[]
  /** 各爻是否动爻（6/9）。 */
  dong: boolean[]
  /** 本卦名（如「天风姤」）。 */
  benGua: string
  /** 变卦名；无动爻时为 null。 */
  bianGua: string | null
}

// 三爻（自下而上，阳=1 阴=0）→ 八卦：b1<<2 | b2<<1 | b3。
const TRIGRAMS = ['坤', '艮', '坎', '巽', '震', '离', '兑', '乾'] as const

/** 下卦+上卦 → 卦名（含八纯卦「乾为天」式命名）。 */
const GUAS: Record<string, string> = {
  乾乾: '乾为天',
  乾兑: '泽天夬',
  乾坎: '水天需',
  乾坤: '地天泰',
  乾巽: '风天小畜',
  乾离: '火天大有',
  乾艮: '山天大畜',
  乾震: '雷天大壮',
  兑乾: '天泽履',
  兑兑: '兑为泽',
  兑坎: '水泽节',
  兑坤: '地泽临',
  兑巽: '风泽中孚',
  兑离: '火泽睽',
  兑艮: '山泽损',
  兑震: '雷泽归妹',
  坎乾: '天水讼',
  坎兑: '泽水困',
  坎坎: '坎为水',
  坎坤: '地水师',
  坎巽: '风水涣',
  坎离: '火水未济',
  坎艮: '山水蒙',
  坎震: '雷水解',
  坤乾: '天地否',
  坤兑: '泽地萃',
  坤坎: '水地比',
  坤坤: '坤为地',
  坤巽: '风地观',
  坤离: '火地晋',
  坤艮: '山地剥',
  坤震: '雷地豫',
  巽乾: '天风姤',
  巽兑: '泽风大过',
  巽坎: '水风井',
  巽坤: '地风升',
  巽巽: '巽为风',
  巽离: '火风鼎',
  巽艮: '山风蛊',
  巽震: '雷风恒',
  离乾: '天火同人',
  离兑: '泽火革',
  离坎: '水火既济',
  离坤: '地火明夷',
  离巽: '风火家人',
  离离: '离为火',
  离艮: '山火贲',
  离震: '雷火丰',
  艮乾: '天山遁',
  艮兑: '泽山咸',
  艮坎: '水山蹇',
  艮坤: '地山谦',
  艮巽: '风山渐',
  艮离: '火山旅',
  艮艮: '艮为山',
  艮震: '雷山小过',
  震乾: '天雷无妄',
  震兑: '泽雷随',
  震坎: '水雷屯',
  震坤: '地雷复',
  震巽: '风雷益',
  震离: '火雷噬嗑',
  震艮: '山雷颐',
  震震: '震为雷',
}

function trigram(y0: boolean, y1: boolean, y2: boolean): string {
  return TRIGRAMS[(y0 ? 4 : 0) + (y1 ? 2 : 0) + (y2 ? 1 : 0)]
}

function yang(value: number): boolean {
  // 7 少阳、9 老阳为阳；6 老阴、8 少阴为阴。
  return value === 7 || value === 9
}

function nameOf(yaos: boolean[]): string {
  const down = trigram(yaos[0], yaos[1], yaos[2])
  const up = trigram(yaos[3], yaos[4], yaos[5])
  return GUAS[`${down}${up}`] ?? ''
}

/** casting → 本卦/变卦（免费展示用；解卦（六亲/纳甲）在 App 侧）。 */
export function readCasting(casting: LiuyaoCasting): YiGuaReading {
  const yaos = (casting.yaos ?? []).slice(0, 6) as number[]
  const dong = yaos.map(v => v === 6 || v === 9)
  const ben = yaos.map(yang)
  const bian = ben.map((y, i) => (dong[i] ? !y : y))
  const hasDong = dong.some(Boolean)
  return {
    yaos,
    dong,
    benGua: nameOf(ben),
    bianGua: hasDong ? nameOf(bian) : null,
  }
}
