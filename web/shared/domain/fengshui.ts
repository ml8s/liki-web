import {
  type BazhaiChart,
  type BazhaiLayout,
  type XuankongChart,
  engineBazhaiChart,
  engineBazhaiLayout,
  engineXuankongChart,
} from '../api/engine/fengshui'

export type { BazhaiChart, BazhaiLayout, XuankongChart }

/** Birth year + gender -> Bazhai chart. */
export function bazhaiChartFor(birthYear: number, gender: string): Promise<BazhaiChart> {
  return engineBazhaiChart(birthYear, gender)
}

/** Period date + sitting mountain -> Xuankong flying-star chart. */
export function xuankongChartFor(
  periodDate: string,
  zuoShan: number,
): Promise<XuankongChart> {
  return engineXuankongChart(periodDate, zuoShan, (zuoShan + 12) % 24)
}

/** Ming Gua + door/master/stove Gua -> Bazhai layout assessment. */
export function bazhaiLayoutFor(
  mingGua: string,
  doorGua: string,
  masterGua: string,
  stoveGua: string,
): Promise<BazhaiLayout> {
  return engineBazhaiLayout(mingGua, doorGua, masterGua, stoveGua)
}
