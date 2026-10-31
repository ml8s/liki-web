import { computed, reactive } from 'vue'
import type { MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import type { ConsultationContext } from '@shared/domain/consultation'
import type { IntentGroupId } from '@shared/domain/intent-flow'
import type { LiuyaoCasting } from '@shared/api/engine/liuyao'
import type { LiuyaoChartResult } from '@shared/api/engine/liuyao'
import type { ArtifactPayloadByKind } from '@/components/flow/artifacts/types'
import {
  buildArtifactDescriptors,
  findArtifact,
  type AnyArtifactDescriptor,
} from '@/components/flow/artifacts'

type ArtifactDesktopMode = 'peek' | 'pinned'

interface ArtifactChannelState {
  selectedId: string | null
  open: boolean
  desktopMode: ArtifactDesktopMode
}

function isLiuyaoCasting(value: unknown): value is LiuyaoCasting {
  if (!value || typeof value !== 'object') return false
  const casting = value as Partial<LiuyaoCasting>
  return casting.order === 'bottom_up'
    && Array.isArray(casting.yaos)
    && casting.yaos.length === 6
    && Array.isArray(casting.dong_yao)
    && typeof casting.casting_id === 'string'
    && casting.casting_id.length > 0
}

/**
 * A consultation context stores deterministic receipts, not chat prose.
 * Only such receipts may become artifacts; the model answer is never parsed.
 */
function artifactsFromConsultation(
  groupId: IntentGroupId,
  context: ConsultationContext | null | undefined,
): AnyArtifactDescriptor[] {
  if (!context?.casting || !isLiuyaoCasting(context.casting)) return []
  return buildArtifactDescriptors(groupId, {
    liuyao: { casting: context.casting },
  })
}

export function useIntentArtifacts(options: {
  channelKey: MaybeRefOrGetter<string>
  groupId: MaybeRefOrGetter<IntentGroupId>
  context: MaybeRefOrGetter<ConsultationContext | null | undefined>
  casting?: MaybeRefOrGetter<LiuyaoCasting | null>
  chart?: MaybeRefOrGetter<LiuyaoChartResult | null>
  hepan?: MaybeRefOrGetter<ArtifactPayloadByKind['hepan'] | null>
  fengshui?: MaybeRefOrGetter<ArtifactPayloadByKind['fengshui'] | null>
  naming?: MaybeRefOrGetter<ArtifactPayloadByKind['naming'] | null>
}) {
  const channelStates = reactive<Record<string, ArtifactChannelState>>({})

  function stateFor(key: string): ArtifactChannelState {
    channelStates[key] ??= {
      selectedId: null,
      open: false,
      desktopMode: 'peek',
    }
    return channelStates[key]!
  }

  const descriptors = computed(() => {
    const groupId = toValue(options.groupId)
    const casting = toValue(options.casting)
    const chart = toValue(options.chart)
    const hepan = toValue(options.hepan)
    const fengshui = toValue(options.fengshui)
    const naming = toValue(options.naming)

    if (naming) {
      return buildArtifactDescriptors(groupId, { naming })
    }

    if (fengshui) {
      return buildArtifactDescriptors(groupId, { fengshui })
    }

    if (hepan) {
      return buildArtifactDescriptors(groupId, { hepan })
    }

    // A casting receipt alone is not a chart artifact; wait for the chart so a
    // failed assembly never fabricates a ready artifact.
    if (casting && chart) {
      return buildArtifactDescriptors(groupId, {
        liuyao: {
          casting,
          chart,
        },
      })
    }

    return artifactsFromConsultation(groupId, toValue(options.context))
  })

  const state = computed(() => stateFor(toValue(options.channelKey)))
  const selectedArtifact = computed(() =>
    findArtifact(descriptors.value, state.value.selectedId))
  const selectedId = computed(() =>
    selectedArtifact.value?.id ?? null)
  const open = computed(() => state.value.open)
  const desktopMode = computed(() => state.value.desktopMode)

  function toggleArtifact(id: string) {
    if (!descriptors.value.some(descriptor => descriptor.id === id)) return
    const current = state.value
    if (current.open && current.selectedId === id) {
      current.open = false
      return
    }
    current.selectedId = id
    current.open = true
  }

  function closeArtifact() {
    state.value.open = false
  }

  function setDesktopMode(mode: ArtifactDesktopMode) {
    state.value.desktopMode = mode
  }

  return {
    artifacts: descriptors,
    open,
    selectedArtifact,
    selectedId,
    desktopMode,
    closeArtifact,
    setDesktopMode,
    toggleArtifact,
  }
}
