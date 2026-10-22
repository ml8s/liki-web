import { type BaziYongShenResult, engineBaziYongShen } from '../api/engine/bazi'

export type NamingAssessment = BaziYongShenResult

/** Bazi chart -> deterministic naming assessment from counsel MCP. */
export function namingAssessmentFor(chart: unknown): Promise<NamingAssessment> {
  return engineBaziYongShen(chart)
}
