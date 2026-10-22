import {
  type BaziChartData,
  type BaziFullChartData,
  engineBaziChart,
  engineBaziFullChart,
  engineBaziLiunian,
  type BaziLiuNianOutput,
} from '../api/engine/bazi'
import {
  type ZiweiChart,
  type ZiweiDaXianItem,
  engineZiweiChart,
  engineZiweiDaXian,
  engineZiweiFullChart,
  engineZiweiLiuNian,
  type ZiweiLiuNianData,
} from '../api/engine/ziwei'
import type { BirthInput } from './birth'
import { chartInputFromBirth } from './birth'
import type { ChartInput } from './time'

export async function baziChartFor(birth: BirthInput): Promise<BaziChartData> {
  const input = await chartInputFromBirth(birth)
  return engineBaziChart({ ...input, gender: birth.gender })
}

export function baziChartForChartInput(
  input: ChartInput & { gender: string },
): Promise<BaziChartData> {
  return engineBaziChart(input)
}

export interface BaziFullChart {
  chart: BaziChartData
  full: BaziFullChartData
  input: ChartInput
}

export async function baziFullChartFor(birth: BirthInput): Promise<BaziFullChart> {
  const input = await chartInputFromBirth(birth)
  const chart = await engineBaziChart({ ...input, gender: birth.gender })
  const full = await engineBaziFullChart(chart)
  return { chart, full, input }
}

export interface NatalChart {
  input: ChartInput
  bazi: BaziChartData
  baziDetail: BaziFullChartData
  ziwei: ZiweiChart
  ziweiDaXian: ZiweiDaXianItem[]
}

export async function natalChartForChartInput(
  input: ChartInput & { gender: string },
): Promise<NatalChart> {
  const [bazi, ziweiBase] = await Promise.all([
    engineBaziChart(input),
    engineZiweiChart(input),
  ])
  const [baziDetail, ziwei] = await Promise.all([
    engineBaziFullChart(bazi),
    engineZiweiFullChart(ziweiBase),
  ])
  const ziweiDaXian = await engineZiweiDaXian(ziwei)
  return { input, bazi, baziDetail, ziwei, ziweiDaXian }
}

/** Birth facts -> complete natal chart (Bazi + Ziwei). */
export async function natalChartFor(birth: BirthInput): Promise<NatalChart> {
  const input = await chartInputFromBirth(birth)
  return natalChartForChartInput({ ...input, gender: birth.gender })
}

export type {
  BaziChartData,
  BaziFullChartData,
  BaziLiuNianOutput,
  ZiweiChart,
  ZiweiLiuNianData,
  ZiweiDaXianItem,
}

/** Natal Bazi chart + target year -> annual chart. */
export function baziLiunianFor(
  chart: BaziChartData,
  year: number,
): Promise<BaziLiuNianOutput> {
  return engineBaziLiunian(chart, year)
}

/** Full Ziwei chart + lunar year -> annual palace chart. */
export function ziweiLiunianFor(
  chart: ZiweiChart,
  lunarYear: number,
): Promise<ZiweiLiuNianData> {
  return engineZiweiLiuNian(chart, lunarYear)
}
