import {
  engineBaziChart,
  engineBaziLiuri,
  type BaziChartData,
  type BaziLiuRiOutput,
} from '../api/engine/bazi'
import {
  engineZiweiChart,
  engineZiweiLiuri,
  type ZiweiChart,
  type ZiweiLiuRiData,
} from '../api/engine/ziwei'
import { chartInputFromBirth, type BirthInput } from './birth'
import { standardTimeForDate } from './time'

/** A date-selection flow-day query always owns both event date and birth facts. */
export interface FlowDayInput {
  birth: BirthInput
  selectedDate: string
}

function selectedStandardTime(birth: BirthInput, selectedDate: string): string {
  return standardTimeForDate(selectedDate, birth.location.timezone)
}

/** Birth facts + selected day -> Bazi chart -> deterministic Bazi flow day. */
export async function baziFlowDayFor(
  input: FlowDayInput,
): Promise<{ chart: BaziChartData; day: BaziLiuRiOutput }> {
  const chartInput = await chartInputFromBirth(input.birth)
  const chart = await engineBaziChart({ ...chartInput, gender: input.birth.gender })
  const [day] = await engineBaziLiuri(
    chart,
    selectedStandardTime(input.birth, input.selectedDate),
    chartInput.longitude,
  )
  if (!day) throw new RangeError(`Bazi flow day ${input.selectedDate} was unavailable`)
  return { chart, day }
}

/** Birth facts + selected day -> Ziwei chart -> deterministic Ziwei flow day. */
export async function ziweiFlowDayFor(
  input: FlowDayInput,
): Promise<{ chart: ZiweiChart; day: ZiweiLiuRiData }> {
  const chartInput = await chartInputFromBirth(input.birth)
  const chart = await engineZiweiChart({ ...chartInput, gender: input.birth.gender })
  const [day] = await engineZiweiLiuri(
    chart,
    selectedStandardTime(input.birth, input.selectedDate),
  )
  if (!day) throw new RangeError(`Ziwei flow day ${input.selectedDate} was unavailable`)
  return { chart, day }
}
