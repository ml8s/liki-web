import { engineHuangliDays, type HuangliDay } from '../api/engine/huangli'

export type HuangliDays = HuangliDay[]

/** Explicit event + Gregorian range -> daily almanac facts. */
export function huangliDaysFor(
  event: string,
  startDate: string,
  count: number,
): Promise<HuangliDays> {
  return engineHuangliDays(event, startDate, count)
}
