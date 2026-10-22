import {
  engineBaziBond,
  engineBaziChart,
  type BaziBondResult,
  type BaziChartData,
} from '../api/engine/bazi'
import type { BirthInput } from './birth'
import { chartInputFromBirth } from './birth'
import type { ChartInput } from './time'

export interface Hepan {
  chartA: BaziChartData
  chartB: BaziChartData
  bond: BaziBondResult
}

export async function hepanForChartInputs(
  a: ChartInput & { gender: string },
  b: ChartInput & { gender: string },
): Promise<Hepan> {
  const [chartA, chartB] = await Promise.all([
    engineBaziChart(a),
    engineBaziChart(b),
  ])
  const bond = await engineBaziBond(chartA, chartB)
  return { chartA, chartB, bond }
}

/** Two complete birth facts -> Bazi compatibility chart. */
export async function hepanFor(a: BirthInput, b: BirthInput): Promise<Hepan> {
  const [inputA, inputB] = await Promise.all([
    chartInputFromBirth(a),
    chartInputFromBirth(b),
  ])
  return hepanForChartInputs(
    { ...inputA, gender: a.gender },
    { ...inputB, gender: b.gender },
  )
}
