import {
  type BazhaiChart,
  type BazhaiLayout,
  type XuankongChart,
  engineBazhaiChart,
  engineBazhaiLayout,
  engineXuankongChart,
} from '../api/engine/fengshui'

export type { BazhaiChart, BazhaiLayout, XuankongChart }

/** Birth standard time + gender -> Bazhai chart. */
export function bazhaiChartFor(standardTime: string, gender: string): Promise<BazhaiChart> {
  return engineBazhaiChart(standardTime, gender)
}

/** Already-normalized period standard time + sitting mountain -> Xuankong chart. */
export function xuankongChartFor(
  standardTime: string,
  zuoShan: number,
): Promise<XuankongChart> {
  return engineXuankongChart(standardTime, zuoShan, (zuoShan + 12) % 24)
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
