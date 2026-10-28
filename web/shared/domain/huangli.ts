import { engineHuangliDays, type HuangliCount, type HuangliDay } from '../api/engine/huangli'
import { standardTimeForDate } from './time'

export type HuangliDays = HuangliDay[]

/** Explicit event + Gregorian date range -> daily almanac facts. */
export function huangliDaysFor(
  event: string,
  startDate: string,
  count: number,
): Promise<HuangliDays> {
  if (!Number.isInteger(count) || count < 1 || count > 30) {
    throw new RangeError(`Huangli count ${count} is outside 1..30`)
  }
  return engineHuangliDays(event, standardTimeForDate(startDate), count as HuangliCount)
}
