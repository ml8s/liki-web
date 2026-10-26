import { existsSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import {
  STANDARD_ENGINE_RESULT_PUBLIC_PROPS,
  engineResultByMethod,
  engineResultCatalog,
  pendingEngineResults,
  sourceBackedEngineResults,
} from '@shared/components/engine/catalog'
import type {
  EngineResultPanelProps,
  EngineResultPanelPropsFor,
} from '@shared/components/engine/contract'
import type {
  BaziBondResult,
  BaziChartData,
  BaziFullChartData,
  BaziLiuNianOutput,
  BaziYongShenResult,
} from '@shared/api/engine/bazi'
import type { BazhaiChart, BazhaiLayout, XuankongChart } from '@shared/api/engine/fengshui'
import type { HuangliDay } from '@shared/api/engine/huangli'
import type { LiuyaoCasting, LiuyaoChartResult } from '@shared/api/engine/liuyao'
import type { ZiweiChart, ZiweiDaXianItem, ZiweiLiuNianData } from '@shared/api/engine/ziwei'

type SameType<Actual, Expected> =
  (<Value>() => Value extends Actual ? 1 : 2) extends
  (<Value>() => Value extends Expected ? 1 : 2) ? true : false

type ExactStandardProps<Result> = EngineResultPanelProps<Result>

type ContractAssertions = [
  SameType<EngineResultPanelPropsFor<'bazi_chart'>, ExactStandardProps<BaziChartData>>,
  SameType<EngineResultPanelPropsFor<'bazi_fullchart'>, ExactStandardProps<BaziFullChartData>>,
  SameType<EngineResultPanelPropsFor<'bazi_liunian'>, ExactStandardProps<BaziLiuNianOutput>>,
  SameType<EngineResultPanelPropsFor<'bazi_yongshen'>, ExactStandardProps<BaziYongShenResult>>,
  SameType<
    EngineResultPanelPropsFor<'bazi_bond'>,
    ExactStandardProps<BaziBondResult> & {
      chartA: BaziChartData
      chartB: BaziChartData
      nameA?: string
      nameB?: string
    }
  >,
  SameType<EngineResultPanelPropsFor<'ziwei_chart'>, ExactStandardProps<ZiweiChart>>,
  SameType<EngineResultPanelPropsFor<'ziwei_fullchart'>, ExactStandardProps<ZiweiChart>>,
  SameType<EngineResultPanelPropsFor<'ziwei_daxian'>, ExactStandardProps<ZiweiDaXianItem[]>>,
  SameType<EngineResultPanelPropsFor<'ziwei_liunian'>, ExactStandardProps<ZiweiLiuNianData>>,
  SameType<EngineResultPanelPropsFor<'liuyao_qigua'>, ExactStandardProps<LiuyaoCasting>>,
  SameType<EngineResultPanelPropsFor<'liuyao_chart'>, ExactStandardProps<LiuyaoChartResult>>,
  SameType<EngineResultPanelPropsFor<'huangli_days'>, ExactStandardProps<HuangliDay[]>>,
  SameType<
    EngineResultPanelPropsFor<'bazhai_chart'>,
    ExactStandardProps<BazhaiChart> & { layout?: BazhaiLayout | null }
  >,
  SameType<
    EngineResultPanelPropsFor<'bazhai_layout'>,
    ExactStandardProps<BazhaiLayout>
  >,
  SameType<EngineResultPanelPropsFor<'xuankong_chart'>, ExactStandardProps<XuankongChart>>,
]

const contractAssertions: ContractAssertions = [
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
  true,
]

// Vitest runs with web/app as its project root; catalog paths are repo-rooted.
const repositoryRoot = resolve(process.cwd(), '../..')

function repositoryPath(path: string): string {
  return join(repositoryRoot, path)
}

describe('engine result catalog contract', () => {
  it('maps every panel method to the exact shared DTO-backed props contract', () => {
    expect(contractAssertions).toHaveLength(15)
    expect(contractAssertions.every(isTrue)).toBe(true)
  })

  it('has unique methods and canonical target paths', () => {
    const methods = engineResultCatalog.map(entry => entry.method)
    const targetPaths = engineResultCatalog
      .map(entry => entry.targetPath)
      .filter(path => path !== null)

    expect(new Set(methods).size).toBe(methods.length)
    expect(new Set(targetPaths).size).toBe(targetPaths.length)
  })

  it('freezes provenance, baselines, and implementation paths for source-backed panels', () => {
    expect(sourceBackedEngineResults.length).toBe(16)

    for (const entry of sourceBackedEngineResults) {
      expect(entry.sourcePaths.length, entry.method).toBeGreaterThan(0)
      expect(entry.baselineTest, entry.method).toBeTypeOf('string')

      for (const sourcePath of entry.sourcePaths) {
        expect(existsSync(repositoryPath(sourcePath)), `${entry.method}: ${sourcePath}`).toBe(true)
      }

      if (entry.kind === 'result-panel') {
        expect(entry.targetPath, entry.method).toMatch(/^web\/shared\/components\/engine\/[\w-]+\/\w+\.vue$/)
        expect(entry.publicProps.slice(0, 3)).toEqual([...STANDARD_ENGINE_RESULT_PUBLIC_PROPS])
      }
      else {
        expect(entry.targetPath, entry.method).toBeNull()
      }

      if (entry.baselineTest) {
        expect(existsSync(repositoryPath(entry.baselineTest)), entry.baselineTest).toBe(true)
      }

      const supportTargetPaths = 'supportTargetPaths' in entry
        ? entry.supportTargetPaths
        : []
      for (const supportPath of supportTargetPaths) {
        expect(supportPath).toMatch(/^web\/shared\/components\/engine\/[\w-]+\/\w+\.vue$/)
      }
    }
  })

  it('declares context props required by the bond result family', () => {
    const entry = engineResultByMethod('bazi_bond')
    expect(entry?.publicProps).toEqual([
      'result',
      'focused?',
      'density?',
      'chartA',
      'chartB',
      'nameA?',
      'nameB?',
    ])
  })

  it('keeps pending methods explicit and prevents implementation by inference', () => {
    expect(pendingEngineResults.map(entry => entry.method)).toEqual(['bazi_liuri', 'ziwei_liuri'])

    for (const entry of pendingEngineResults) {
      expect(entry.dto).toBe('pending')
      expect(entry.sourcePaths).toEqual([])
      expect(entry.targetPath).toBeNull()
      expect(entry.publicProps).toEqual([])
      expect(entry.knownGaps?.length ?? 0).toBeGreaterThan(0)
      expect(entry.baselineTest).toBeNull()
    }
  })
})

function isTrue(value: boolean): value is true {
  return value === true
}
