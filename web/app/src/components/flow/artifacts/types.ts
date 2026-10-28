import type { AppDomainId } from '@shared/domain/flow'
import type {
  BaziBondResult,
  BaziChartData,
  BaziFullChartData,
  BaziLiuNianOutput,
  BaziYongShenResult,
} from '@shared/api/engine/bazi'
import type { HuangliDay } from '@shared/api/engine/huangli'
import type { LiuyaoCasting, LiuyaoChartResult } from '@shared/api/engine/liuyao'
import type { BazhaiChart, BazhaiLayout, XuankongChart } from '@shared/api/engine/fengshui'
import type { ZiweiChart, ZiweiDaXianItem } from '@shared/api/engine/ziwei'

export type ArtifactKind =
  | 'ziwei'
  | 'bazi'
  | 'hepan'
  | 'liuyao'
  | 'huangli'
  | 'fengshui'
  | 'naming'

export type ArtifactState = 'ready' | 'pending-backend' | 'unavailable'

export interface ArtifactPayloadByKind {
  ziwei: {
    chart: ZiweiChart
    daXian?: ZiweiDaXianItem[]
  }
  bazi: {
    chart: BaziChartData
    full?: BaziFullChartData | null
    liunian?: BaziLiuNianOutput | null
  }
  hepan: {
    chartA: BaziChartData
    chartB: BaziChartData
    bond: BaziBondResult
    nameA: string
    nameB?: string
  }
  liuyao: {
    casting: LiuyaoCasting
    chart?: LiuyaoChartResult | null
  }
  huangli: {
    days: HuangliDay[]
  }
  fengshui: {
    bazhai?: BazhaiChart | null
    layout?: BazhaiLayout | null
    xuankong?: XuankongChart | null
  }
  naming: {
    assessment: BaziYongShenResult
  }
}

export interface ArtifactDescriptor<Kind extends ArtifactKind = ArtifactKind> {
  id: string
  domainId: AppDomainId
  kind: Kind
  titleKey: string
  state: ArtifactState
  payload?: ArtifactPayloadByKind[Kind]
}

export type AnyArtifactDescriptor = ArtifactDescriptor<ArtifactKind>

type ReadyArtifact = {
  [Kind in ArtifactKind]: ArtifactDescriptor<Kind> & { state: 'ready'; payload: ArtifactPayloadByKind[Kind] }
}[ArtifactKind]

export type AnyReadyArtifact = ReadyArtifact

export function isReadyArtifact(
  descriptor: AnyArtifactDescriptor,
): descriptor is AnyReadyArtifact {
  return descriptor.state === 'ready' && descriptor.payload !== undefined
}

export function createReadyArtifact<Kind extends ArtifactKind>(
  descriptor: Omit<ArtifactDescriptor<Kind>, 'state'> & {
    payload: ArtifactPayloadByKind[Kind]
  },
): ArtifactDescriptor<Kind> {
  return { ...descriptor, state: 'ready' }
}
