import type {
  BaziBondResult,
  BaziChartData,
  BaziFullChartData,
  BaziLiuNianOutput,
  BaziLiuRiOutput,
  BaziYongShenResult,
} from '../../api/engine/bazi'
import type { CityCandidate } from '../../api/engine/city'
import type { BazhaiChart, BazhaiLayout, XuankongChart } from '../../api/engine/fengshui'
import type { HuangliDay } from '../../api/engine/huangli'
import type { LiuyaoCasting, LiuyaoChartResult } from '../../api/engine/liuyao'
import type {
  ZiweiChart,
  ZiweiDaXianItem,
  ZiweiLiuNianData,
  ZiweiLiuRiData,
} from '../../api/engine/ziwei'

export type EngineResultDensity = 'compact' | 'full'

/** Common presentational contract for a deterministic engine result panel. */
export interface EngineResultPanelProps<Result> {
  result: Result
  /** Stable domain identifier, such as a palace, pillar, or day key. */
  focused?: string
  density?: EngineResultDensity
}

/**
 * Compile-time mapping from an engine method to its web DTO. Result panels must
 * derive their props through `EngineResultPanelPropsFor`; cloning these shapes
 * is a contract violation.
 */
export interface EngineResultDtoMap {
  city_coords: CityCandidate
  bazi_chart: BaziChartData
  bazi_fullchart: BaziFullChartData
  bazi_liunian: BaziLiuNianOutput
  bazi_liuri: BaziLiuRiOutput
  bazi_yongshen: BaziYongShenResult
  bazi_bond: BaziBondResult
  ziwei_chart: ZiweiChart
  ziwei_fullchart: ZiweiChart
  ziwei_daxian: ZiweiDaXianItem[]
  ziwei_liunian: ZiweiLiuNianData
  ziwei_liuri: ZiweiLiuRiData
  liuyao_qigua: LiuyaoCasting
  liuyao_chart: LiuyaoChartResult
  huangli_days: HuangliDay[]
  bazhai_chart: BazhaiChart
  bazhai_layout: BazhaiLayout
  xuankong_chart: XuankongChart
}

export type SourceBackedEngineResultMethod = keyof EngineResultDtoMap

export type EngineResultPanelMethod = Exclude<
  SourceBackedEngineResultMethod,
  'city_coords'
>

/**
 * Methods behind engine/counsel boundaries that have no exposed web DTO yet.
 * They are catalog gaps and must not be implemented by inference.
 */
export type EngineResultMethod = SourceBackedEngineResultMethod

/** Bond projection needs the two source charts to render both pillars. */
export interface BaziBondPanelContextProps {
  chartA: BaziChartData
  chartB: BaziChartData
  nameA?: string
  nameB?: string
}

export type EngineResultPanelPropsFor<Method extends EngineResultPanelMethod> =
  Method extends 'bazi_bond'
    ? EngineResultPanelProps<EngineResultDtoMap[Method]> & BaziBondPanelContextProps
    : Method extends 'bazhai_chart'
      ? EngineResultPanelProps<EngineResultDtoMap[Method]> & BazhaiChartPanelContextProps
      : EngineResultPanelProps<EngineResultDtoMap[Method]>

/**
 * Vue SFC's compiler cannot lower a generic conditional type in
 * `defineProps`. These aliases are the canonical runtime-safe public contracts;
 * the generic helper remains for pure TypeScript integration assertions.
 */
export type BaziChartPanelProps = EngineResultPanelProps<BaziChartData>
export type BaziFullChartPanelProps = EngineResultPanelProps<BaziFullChartData>
export type BaziLiuNianPanelProps = EngineResultPanelProps<BaziLiuNianOutput>
export type BaziLiuRiPanelProps = EngineResultPanelProps<BaziLiuRiOutput>
export type BaziYongShenPanelProps = EngineResultPanelProps<BaziYongShenResult>
export type BaziBondPanelProps =
  EngineResultPanelProps<BaziBondResult> & BaziBondPanelContextProps
export type ZiweiChartPanelProps = EngineResultPanelProps<ZiweiChart>
export type ZiweiFullChartPanelProps = EngineResultPanelProps<ZiweiChart>
export type ZiweiDaXianPanelProps = EngineResultPanelProps<ZiweiDaXianItem[]>
export type ZiweiLiuNianPanelProps = EngineResultPanelProps<ZiweiLiuNianData>
export type ZiweiLiuRiPanelProps = EngineResultPanelProps<ZiweiLiuRiData>
export type LiuyaoCastingPanelProps = EngineResultPanelProps<LiuyaoCasting>
export type LiuyaoChartPanelProps = EngineResultPanelProps<LiuyaoChartResult>
export type HuangliDaysPanelProps = EngineResultPanelProps<HuangliDay[]>
/** The optionally loaded layout annotates the chart-family direction grid. */
export interface BazhaiChartPanelContextProps {
  layout?: BazhaiLayout | null
}
export type BazhaiChartPanelProps =
  EngineResultPanelProps<BazhaiChart> & BazhaiChartPanelContextProps
export type BazhaiLayoutPanelProps = EngineResultPanelProps<BazhaiLayout>
export type XuankongChartPanelProps = EngineResultPanelProps<XuankongChart>

/** Private member of the `huangli_days` result family, not another method. */
export interface HuangliDayDetailProps {
  day: HuangliDay
  focused?: string
  density?: EngineResultDensity
}
