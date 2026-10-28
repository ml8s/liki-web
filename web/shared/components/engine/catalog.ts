import type {
  EngineResultMethod,
  EngineResultPanelMethod,
} from './contract'

export type {
  BaziBondPanelContextProps,
  EngineResultDensity,
  EngineResultDtoMap,
  EngineResultPanelProps,
  EngineResultPanelPropsFor,
  HuangliDayDetailProps,
  PendingEngineMethod,
  SourceBackedEngineResultMethod,
} from './contract'

/** Frozen public props for a standard deterministic result panel. */
export const STANDARD_ENGINE_RESULT_PUBLIC_PROPS = [
  'result',
  'focused?',
  'density?',
] as const

export type EngineCatalogKind = 'result-panel' | 'input-support'
export type EngineCatalogStatus = 'source-backed' | 'pending-contract'
export type Wave4ExtractionLane =
  | 'B1' | 'B2' | 'B3' | 'B4' | 'B5' | 'B6' | 'B7' | 'B8' | 'B9' | 'B10'

export interface EngineResultCatalogEntry {
  readonly method: EngineResultMethod
  /** Stable DTO name. This catalog is an inventory; it does not clone DTOs. */
  readonly dto: string
  readonly kind: EngineCatalogKind
  readonly target: string
  /**
   * Canonical implementation path for the public panel. A pending contract has
   * no target path because inferring it from another method is prohibited.
   */
  readonly targetPath: string | null
  /** Private sections that belong to the same result family. */
  readonly supportTargetPaths?: readonly string[]
  /** Human-auditable public props; the executable contract is in contract.ts. */
  readonly publicProps: readonly string[]
  readonly sourcePaths: readonly string[]
  readonly sourceProjection: string
  readonly compatibilityAliases?: readonly string[]
  readonly baselineTest: string | null
  readonly wave4Lane?: Wave4ExtractionLane
  readonly knownGaps?: readonly string[]
  readonly status: EngineCatalogStatus
}

export const engineResultCatalog = [
  {
    method: 'city_coords',
    dto: 'CityCandidate',
    kind: 'input-support',
    target: 'CityCoordinateInputSupport',
    targetPath: null,
    publicProps: [],
    sourcePaths: [
      'web/shared/api/engine/city.ts',
      'web/shared/utils/citySearch.ts',
      'web/shared/components/form/fields/RegionAutocompleteField.vue',
    ],
    sourceProjection: 'Canonical city candidate matching with longitude, latitude, timezone, and longitude range',
    baselineTest: 'web/app/src/tests/city-search.test.ts',
    status: 'source-backed',
  },
  {
    method: 'bazi_chart',
    dto: 'BaziChartData',
    kind: 'result-panel',
    target: 'BaziChartPanel',
    targetPath: 'web/shared/components/engine/bazi/BaziChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?', 'bazhai?'],
    sourcePaths: [
      'web/shared/components/charts/bazi/BaziPillars.vue',
      'web/shared/components/charts/BaziPanel.vue',
    ],
    sourceProjection: 'Four pillars with stems, branches, nayin, and pillar labels',
    compatibilityAliases: ['BaziPillars'],
    baselineTest: 'web/e2e/journeys/natal-tool.spec.ts',
    wave4Lane: 'B1',
    status: 'source-backed',
  },
  {
    method: 'bazi_fullchart',
    dto: 'BaziFullChartData',
    kind: 'result-panel',
    target: 'BaziFullChartPanel',
    targetPath: 'web/shared/components/engine/bazi/BaziFullChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/BaziPanel.vue',
      'web/shared/components/charts/bazi/BaziPillars.vue',
      'web/shared/components/charts/bazi/BaziRelations.vue',
      'web/shared/components/charts/bazi/BaziCycles.vue',
      'web/shared/components/charts/bazi/BaziLiunianDetail.vue',
      'web/shared/components/charts/bazi/BaziYongShen.vue',
    ],
    sourceProjection: 'Full Bazi composition: pillars, relations, cycles, and full-chart sections in current section order',
    compatibilityAliases: ['BaziPanel'],
    baselineTest: 'web/e2e/journeys/natal-tool.spec.ts',
    wave4Lane: 'B1',
    status: 'source-backed',
  },
  {
    method: 'bazi_liunian',
    dto: 'BaziLiuNianOutput',
    kind: 'result-panel',
    target: 'BaziLiuNianPanel',
    targetPath: 'web/shared/components/engine/bazi/BaziLiuNianPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/bazi/BaziLiunianDetail.vue',
      'web/app/src/components/birth-profiles/ChartPreview.vue',
    ],
    sourceProjection: 'Annual shensha, fuyin/fanyin, and natal/dayun interaction chip groups',
    compatibilityAliases: ['BaziLiunianDetail'],
    baselineTest: 'web/e2e/journeys/liunian-tool.spec.ts',
    wave4Lane: 'B2',
    status: 'source-backed',
  },
  {
    method: 'bazi_yongshen',
    dto: 'BaziYongShenResult',
    kind: 'result-panel',
    target: 'BaziYongShenPanel',
    targetPath: 'web/shared/components/engine/bazi/BaziYongShenPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/YongShenPanel.vue',
      'web/shared/components/charts/bazi/BaziYongShen.vue',
    ],
    sourceProjection: 'Selected yong/xi/ji chips, resolution status, evidence, conflicts, and review note',
    compatibilityAliases: ['YongShenPanel'],
    baselineTest: 'web/e2e/journeys/naming-tool.spec.ts',
    wave4Lane: 'B3',
    status: 'source-backed',
  },
  {
    method: 'bazi_bond',
    dto: 'BaziBondResult',
    kind: 'result-panel',
    target: 'BaziBondPanel',
    targetPath: 'web/shared/components/engine/bazi/BaziBondPanel.vue',
    publicProps: [
      'result',
      'focused?',
      'density?',
      'chartA',
      'chartB',
      'nameA?',
      'nameB?',
    ],
    sourcePaths: [
      'web/shared/components/charts/HepanPanel.vue',
      'web/app/src/views/HepanView.vue',
    ],
    sourceProjection: 'Two-person pillars, day-master relation, branch/nayin relations, cross shensha, and ten-god comparison',
    compatibilityAliases: ['HepanPanel'],
    baselineTest: 'web/e2e/journeys/hepan-tool.spec.ts',
    wave4Lane: 'B4',
    status: 'source-backed',
  },
  {
    method: 'ziwei_chart',
    dto: 'ZiweiChart',
    kind: 'result-panel',
    target: 'ZiweiChartPanel',
    targetPath: 'web/shared/components/engine/ziwei/ZiweiChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: ['web/shared/components/charts/ZiweiPanel.vue'],
    sourceProjection: 'Twelve-palace grid with palace names, stems/branches, major and minor stars, center information, and current highlighting',
    compatibilityAliases: ['ZiweiPanel'],
    baselineTest: 'web/e2e/journeys/natal-tool.spec.ts',
    wave4Lane: 'B5',
    status: 'source-backed',
  },
  {
    method: 'ziwei_fullchart',
    dto: 'ZiweiChart',
    kind: 'result-panel',
    target: 'ZiweiFullChartPanel',
    targetPath: 'web/shared/components/engine/ziwei/ZiweiFullChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: ['web/shared/components/charts/ZiweiPanel.vue'],
    sourceProjection: 'Palace grid plus patterns, sanfang, and kongjie in existing order; daxian remains the separate ziwei_daxian result family',
    compatibilityAliases: ['ZiweiPanel'],
    baselineTest: 'web/e2e/journeys/natal-tool.spec.ts',
    wave4Lane: 'B6',
    status: 'source-backed',
  },
  {
    method: 'ziwei_daxian',
    dto: 'ZiweiDaXianItem[]',
    kind: 'result-panel',
    target: 'ZiweiDaXianPanel',
    targetPath: 'web/shared/components/engine/ziwei/ZiweiDaXianPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/ZiweiPanel.vue',
      'web/app/src/components/birth-profiles/ChartPreview.vue',
    ],
    sourceProjection: 'Daxian name, age range, year range, and current-step marker',
    baselineTest: 'web/e2e/journeys/natal-tool.spec.ts',
    wave4Lane: 'B7',
    status: 'source-backed',
  },
  {
    method: 'ziwei_liunian',
    dto: 'ZiweiLiuNianData',
    kind: 'result-panel',
    target: 'ZiweiLiuNianPanel',
    targetPath: 'web/shared/components/engine/ziwei/ZiweiLiuNianPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/ZiweiPanel.vue',
      'web/app/src/components/birth-profiles/ChartPreview.vue',
    ],
    sourceProjection: 'Annual minggong, zhi, sihua, stars, and annual palace labels',
    baselineTest: 'web/e2e/journeys/liunian-tool.spec.ts',
    wave4Lane: 'B7',
    status: 'source-backed',
  },
  {
    method: 'liuyao_qigua',
    dto: 'LiuyaoCasting',
    kind: 'result-panel',
    target: 'LiuyaoCastingPanel',
    targetPath: 'web/shared/components/engine/liuyao/LiuyaoCastingPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: ['web/shared/components/charts/DivinationCastPanel.vue'],
    sourceProjection: 'Original and changed hexagram names with six bottom-up lines and moving-line cue',
    compatibilityAliases: ['DivinationCastPanel'],
    baselineTest: 'web/e2e/journeys/divination-tool.spec.ts',
    wave4Lane: 'B8',
    status: 'source-backed',
  },
  {
    method: 'liuyao_chart',
    dto: 'LiuyaoChartResult',
    kind: 'result-panel',
    target: 'LiuyaoChartPanel',
    targetPath: 'web/shared/components/engine/liuyao/LiuyaoChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: ['web/app/src/views/DivinationView.vue'],
    sourceProjection: 'Complete liuyao chart: hexagram identity, palace, day/month pillars, lines, changing lines, yongshen, guaci, patterns, and timing candidates',
    baselineTest: 'web/e2e/journeys/divination-tool.spec.ts',
    wave4Lane: 'B8',
    status: 'source-backed',
  },
  {
    method: 'huangli_days',
    dto: 'HuangliDay[]',
    kind: 'result-panel',
    target: 'HuangliDaysPanel',
    targetPath: 'web/shared/components/engine/huangli/HuangliDaysPanel.vue',
    supportTargetPaths: ['web/shared/components/engine/huangli/HuangliDayDetail.vue'],
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/HuangliDaysPanel.vue',
      'web/shared/components/charts/HuangliDayDetail.vue',
    ],
    sourceProjection: 'Recommended/possible/unsuitable day groups followed by expandable day and shichen details',
    baselineTest: 'web/e2e/journeys/huangli-tool.spec.ts',
    wave4Lane: 'B9',
    status: 'source-backed',
  },
  {
    method: 'bazhai_chart',
    dto: 'BazhaiChart',
    kind: 'result-panel',
    target: 'BazhaiChartPanel',
    targetPath: 'web/shared/components/engine/fengshui/BazhaiChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?', 'layout?'],
    sourcePaths: [
      'web/shared/components/charts/MingGuaSection.vue',
      'web/shared/components/charts/LiuNianFlySection.vue',
      'web/shared/components/charts/BazhaiLayoutPanel.vue',
      'web/app/src/views/FengshuiView.vue',
    ],
    sourceProjection: 'Natal Gua, group, direction grid, and annual flying-star section',
    compatibilityAliases: ['MingGuaSection'],
    baselineTest: 'web/e2e/journeys/fengshui-tool.spec.ts',
    wave4Lane: 'B10',
    status: 'source-backed',
  },
  {
    method: 'bazhai_layout',
    dto: 'BazhaiLayout',
    kind: 'result-panel',
    target: 'BazhaiLayoutPanel',
    targetPath: 'web/shared/components/engine/fengshui/BazhaiLayoutPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/BazhaiLayoutPanel.vue',
      'web/app/src/views/FengshuiView.vue',
    ],
    sourceProjection: 'Door/main/stove directions, Gua, youxing, rating, match, and layout verdict table',
    baselineTest: 'web/e2e/journeys/fengshui-tool.spec.ts',
    wave4Lane: 'B10',
    status: 'source-backed',
  },
  {
    method: 'xuankong_chart',
    dto: 'XuankongChart',
    kind: 'result-panel',
    target: 'XuankongChartPanel',
    targetPath: 'web/shared/components/engine/fengshui/XuankongChartPanel.vue',
    publicProps: ['result', 'focused?', 'density?'],
    sourcePaths: [
      'web/shared/components/charts/XuankongPanel.vue',
      'web/app/src/views/FengshuiView.vue',
    ],
    sourceProjection: 'Xuankong period, nine palaces, facing/sitting mountains, and four-situation assessment',
    compatibilityAliases: ['XuankongPanel'],
    baselineTest: 'web/e2e/journeys/fengshui-tool.spec.ts',
    wave4Lane: 'B10',
    status: 'source-backed',
  },
  {
    method: 'bazi_liuri',
    dto: 'pending',
    kind: 'result-panel',
    target: 'BaziLiuRiPanel',
    targetPath: null,
    publicProps: [],
    sourcePaths: [],
    sourceProjection: 'No current web projection; do not infer it from liunian',
    baselineTest: null,
    knownGaps: [
      'Web API adapter and DTO are not exposed',
      'Personalized date selection requires natal chart, dayun, liunian, and selected date',
    ],
    status: 'pending-contract',
  },
  {
    method: 'ziwei_liuri',
    dto: 'pending',
    kind: 'result-panel',
    target: 'ZiweiLiuRiPanel',
    targetPath: null,
    publicProps: [],
    sourcePaths: [],
    sourceProjection: 'No current web projection; do not infer it from liunian',
    baselineTest: null,
    knownGaps: [
      'Web API adapter and DTO are not exposed',
      'Personalized date selection requires the natal Ziwei chart and selected lunar day',
    ],
    status: 'pending-contract',
  },
] as const satisfies readonly EngineResultCatalogEntry[]

export function engineResultPanelPropsAreFrozen(
  method: EngineResultMethod,
): method is EngineResultPanelMethod {
  return engineResultByMethod(method)?.kind === 'result-panel'
    && method !== 'bazi_liuri'
    && method !== 'ziwei_liuri'
}

export const sourceBackedEngineResults = engineResultCatalog.filter(
  entry => entry.status === 'source-backed',
)

export const pendingEngineResults = engineResultCatalog.filter(
  entry => entry.status === 'pending-contract',
)

export function engineResultByMethod(method: EngineResultMethod): EngineResultCatalogEntry | undefined {
  return engineResultCatalog.find(entry => entry.method === method)
}
