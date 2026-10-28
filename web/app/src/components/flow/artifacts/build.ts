import type { AppDomainId } from '@shared/domain/flow'
import type {
  AnyArtifactDescriptor,
  ArtifactDescriptor,
  ArtifactKind,
  ArtifactPayloadByKind,
} from './types'
import { createReadyArtifact } from './types'

type ArtifactSources = {
  [Kind in ArtifactKind]?: ArtifactPayloadByKind[Kind]
}

const TITLE_KEYS: Record<ArtifactKind, string> = {
  ziwei: 'services.natal.label',
  bazi: 'services.natal.label',
  hepan: 'services.hepan.label',
  liuyao: 'services.divination.label',
  huangli: 'services.huangli.label',
  fengshui: 'services.fengshui.label',
  naming: 'services.naming.label',
}

/**
 * The flow only renders artifacts whose deterministic payload actually exists.
 * This adapter intentionally has no fallback sample data.
 */
export function buildArtifactDescriptors(
  domainId: AppDomainId,
  sources: ArtifactSources,
): AnyArtifactDescriptor[] {
  const descriptors: AnyArtifactDescriptor[] = []

  if (sources.ziwei) {
    descriptors.push(createReadyArtifact<'ziwei'>({
      id: `${domainId}:ziwei`,
      domainId,
      kind: 'ziwei',
      titleKey: TITLE_KEYS.ziwei,
      payload: sources.ziwei,
    }))
  }

  if (sources.bazi) {
    descriptors.push(createReadyArtifact<'bazi'>({
      id: `${domainId}:bazi`,
      domainId,
      kind: 'bazi',
      titleKey: TITLE_KEYS.bazi,
      payload: sources.bazi,
    }))
  }

  if (sources.hepan) {
    descriptors.push(createReadyArtifact<'hepan'>({
      id: `${domainId}:hepan`,
      domainId,
      kind: 'hepan',
      titleKey: TITLE_KEYS.hepan,
      payload: sources.hepan,
    }))
  }

  if (sources.liuyao) {
    descriptors.push(createReadyArtifact<'liuyao'>({
      id: `${domainId}:liuyao`,
      domainId,
      kind: 'liuyao',
      titleKey: TITLE_KEYS.liuyao,
      payload: sources.liuyao,
    }))
  }

  if (sources.huangli) {
    descriptors.push(createReadyArtifact<'huangli'>({
      id: `${domainId}:huangli`,
      domainId,
      kind: 'huangli',
      titleKey: TITLE_KEYS.huangli,
      payload: sources.huangli,
    }))
  }

  if (sources.fengshui) {
    descriptors.push(createReadyArtifact<'fengshui'>({
      id: `${domainId}:fengshui`,
      domainId,
      kind: 'fengshui',
      titleKey: TITLE_KEYS.fengshui,
      payload: sources.fengshui,
    }))
  }

  if (sources.naming) {
    descriptors.push(createReadyArtifact<'naming'>({
      id: `${domainId}:naming`,
      domainId,
      kind: 'naming',
      titleKey: TITLE_KEYS.naming,
      payload: sources.naming,
    }))
  }

  return descriptors
}

export function findArtifact(
  descriptors: readonly AnyArtifactDescriptor[],
  id: string | null,
): AnyArtifactDescriptor | null {
  if (!id) return null
  return descriptors.find(descriptor => descriptor.id === id) ?? null
}

export type { ArtifactDescriptor }
