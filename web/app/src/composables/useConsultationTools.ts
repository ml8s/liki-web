import { computed, ref, watch, type MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BirthProfile } from '@/api/birth-profiles'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'
import type { FlowHepanFormValue } from '@/components/flow/forms/FlowHepanForm.vue'
import { birthInputFromFields } from '@shared/components/form/birth-fields'
import type { FengshuiLayoutValue } from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import type { ArtifactPayloadByKind } from '@/components/flow/artifacts/types'
import { useToolRecords } from '@/composables/useToolRecords'
import { birthYearFromDate, chartInputFromBirth, chartInputFromProfileBirth } from '@shared/domain/birth'
import {
  divinationCastingFor,
  divinationChartFor,
  type LiuyaoCasting,
  type LiuyaoChartResult,
} from '@shared/domain/divination'
import { hepanForChartInputs } from '@shared/domain/hepan'
import { baziChartFor } from '@shared/domain/natal'
import { namingAssessmentFor } from '@shared/domain/naming'
import {
  bazhaiChartFor,
  bazhaiLayoutFor,
  xuankongChartFor,
} from '@shared/domain/fengshui'
import type { BirthInput } from '@shared/domain/birth'
import type { ChartInput } from '@shared/domain/time'
import { standardTimeFromInstant, standardTimeForDate } from '@shared/domain/time'
import type { ConsultationContext } from '@shared/domain/consultation'
import { toUserMessage } from '@/utils/errors'
import type { EngineResultFailure } from '@shared/composables/useEngineResult'
import { useGuardedTask } from '@/composables/useGuardedTask'
import { useEngineResult } from '@shared/composables/useEngineResult'

export interface ConsultationToolOptions {
  selectedProfile: MaybeRefOrGetter<BirthProfile | null>
  personLabel: MaybeRefOrGetter<string>
  /** Tool adapters own technical messages; the view decides where to show them. */
  onError: (message: string) => void
  /** Transient positive confirmations (e.g. record saved); the view renders
   *  them through a polite live region. */
  onNotice?: (message: string) => void
}

interface HepanEngineInput {
  a: ChartInput & { gender: string }
  b: ChartInput & { gender: string }
}

interface FengshuiEngineInput {
  birthStandardTime: string
  gender: string
  periodStandardTime: string
  zuoShan: number
  doorGua: string
  masterGua: string
  stoveGua: string
}

/**
 * Owns focused-tool state and deterministic artifact computation. It never
 * creates chat sessions: follow-up routing stays with the consultation view.
 */
export function useConsultationTools(options: ConsultationToolOptions) {
  const { t } = useI18n()
  const reportError = (message: string) => options.onError(message)
  const resourceError = (failure: EngineResultFailure | null, fallback: string) => {
    if (!failure) return ''
    const error = Object.assign(new Error(failure.message), { code: failure.code })
    return toUserMessage(error, fallback)
  }
  // Shared lifecycle for mutation-like actions and input preparation. Queries
  // themselves run through useEngineResult so their errors stay local.
  const castingTask = useGuardedTask({ report: reportError })
  const hepanTask = useGuardedTask({ report: reportError })
  const namingTask = useGuardedTask({ report: reportError })
  const selectedProfile = computed(() => toValue(options.selectedProfile))
  const personLabel = computed(() => toValue(options.personLabel))

  const castingError = ref('')
  const castingQuestion = ref('')
  const casting = ref<LiuyaoCasting | null>(null)
  const castingInstant = ref('')
  const castingLocation = ref<import('@shared/components/divination/types').DivinationCastingFormValue['location']>(null)
  const castingChart = ref<LiuyaoChartResult | null>(null)

  const hepanInput = ref<FlowHepanFormValue | null>(null)
  const hepanEngineInput = ref<HepanEngineInput | null>(null)
  const hepanResource = useEngineResult<HepanEngineInput, Awaited<ReturnType<typeof hepanForChartInputs>>>({
    namespace: 'tool.hepan',
    input: hepanEngineInput,
    load: input => hepanForChartInputs(input.a, input.b),
  })
  const hepanResult = ref<ArtifactPayloadByKind['hepan'] | null>(null)
  watch(() => hepanResource.data.value, hepan => {
    hepanResult.value = hepan
      ? {
          chartA: hepan.chartA,
          chartB: hepan.chartB,
          bond: hepan.bond,
          nameA: personLabel.value,
        }
      : null
  }, { immediate: true })
  const hepanLoading = computed(() =>
    hepanTask.loading.value || hepanResource.status.value === 'loading',
  )
  const hepanError = computed(() =>
    resourceError(hepanResource.error.value, t('services.hepan.failed')),
  )

  const fengshuiInput = ref<FengshuiLayoutValue | null>(null)
  const fengshuiEngineInput = ref<FengshuiEngineInput | null>(null)
  const fengshuiResource = useEngineResult<FengshuiEngineInput, ArtifactPayloadByKind['fengshui']>({
    namespace: 'tool.fengshui',
    input: fengshuiEngineInput,
    load: async input => {
      const [bazhai, xuankong] = await Promise.all([
        bazhaiChartFor(input.birthStandardTime, input.gender),
        xuankongChartFor(input.periodStandardTime, input.zuoShan),
      ])
      const layout = await bazhaiLayoutFor(
        bazhai.ming_gua.gua.name,
        input.doorGua,
        input.masterGua,
        input.stoveGua,
      )
      return { bazhai, layout, xuankong }
    },
  })
  const fengshuiResult = ref<ArtifactPayloadByKind['fengshui'] | null>(null)
  watch(() => fengshuiResource.data.value, data => {
    fengshuiResult.value = data
  }, { immediate: true })
  const fengshuiLoading = computed(() =>
    fengshuiResource.status.value === 'loading',
  )
  const fengshuiError = computed(() =>
    resourceError(fengshuiResource.error.value, t('services.fengshui.failed')),
  )

  const namingEngineInput = ref<BirthInput | null>(null)
  const namingResource = useEngineResult<BirthInput, ArtifactPayloadByKind['naming']>({
    namespace: 'tool.naming',
    input: namingEngineInput,
    load: async input => {
      const chart = await baziChartFor(input)
      return { assessment: await namingAssessmentFor(chart) }
    },
  })
  const namingResult = ref<ArtifactPayloadByKind['naming'] | null>(null)
  watch(() => namingResource.data.value, data => {
    namingResult.value = data
  }, { immediate: true })
  const namingLoading = computed(() =>
    namingTask.loading.value || namingResource.status.value === 'loading',
  )
  const namingError = computed(() =>
    resourceError(namingResource.error.value, t('flow.namingFailed')),
  )

  const {
    save: saveLiuyaoRecord,
  } = useToolRecords('liuyao')
  const {
    save: saveBondRecord,
  } = useToolRecords('bond')
  const {
    save: saveBazhaiRecord,
  } = useToolRecords('bazhai')

  const fengshuiBirthYear = computed(() =>
    birthYearFromDate(selectedProfile.value?.birth.date ?? ''),
  )

  const fengshuiGender = computed(() => {
    const gender = selectedProfile.value?.birth.gender
    return gender === 'female' || gender === 'male' ? gender : ''
  })

  function profileChartInput() {
    const profile = selectedProfile.value
    if (!profile) {
      throw new TypeError('A complete primary birth profile is required')
    }
    return chartInputFromProfileBirth(profile)
  }

  function profileBirthInput() {
    const profile = selectedProfile.value
    const birth = profile?.birth
    const longitude = birth?.location.longitude
    const timezone = birth?.location.timezone
    if (!profile || !birth?.gender || !birth.date || !birth.time
      || typeof longitude !== 'number' || !Number.isFinite(longitude)
      || !timezone) {
      throw new TypeError('A complete birth profile is required')
    }
    return {
      gender: birth.gender,
      date: birth.date,
      time: birth.time,
      location: {
        place: birth.location.place,
        longitude,
        timezone,
        source: 'saved' as const,
      },
    }
  }

  function profileConsultationBirth() {
    const birth = profileBirthInput()
    return {
      gender: birth.gender,
      date: birth.date,
      time: birth.time,
      location: {
        longitude: birth.location.longitude,
        timezone: birth.location.timezone,
      },
    }
  }

  function computeHepan(value: FlowHepanFormValue) {
    return hepanTask.run(async () => {
      hepanResource.reset()
      const otherBirth = birthInputFromFields(value.secondBirth)
      const otherInput = await chartInputFromBirth(otherBirth)
      hepanInput.value = value
      hepanEngineInput.value = {
        a: profileChartInput(),
        b: { ...otherInput, gender: otherBirth.gender },
      }
    }, t('services.hepan.failed'))
  }

  function saveHepanRecord() {
    const input = hepanInput.value
    const profile = selectedProfile.value
    if (!hepanResult.value || !input || !profile) return
    void saveBondRecordAndAnnounce(
      t('records.titleBond', { a: personLabel.value, b: input.secondBirth.date }),
      {
        profile_id: profile.id,
        b: { ...input.secondBirth },
        third_party_consent: true,
      },
    )
  }

  function computeFengshui(value: FengshuiLayoutValue) {
    const profile = selectedProfile.value
    const birthYear = fengshuiBirthYear.value
    if (!profile || birthYear === null || !profile.birth.date) {
      options.onError(t('services.fengshui.invalidBirthYear'))
      return
    }
    let birthStandardTime: string
    try {
      birthStandardTime = chartInputFromProfileBirth(profile).standard_time
    } catch {
      options.onError(t('services.fengshui.invalidBirthYear'))
      return
    }

    fengshuiResource.reset()
    fengshuiInput.value = value
    fengshuiEngineInput.value = {
      birthStandardTime,
      gender: profile.birth.gender,
      periodStandardTime: standardTimeForDate(value.periodDate),
      zuoShan: value.zuoShan,
      doorGua: value.doorGua,
      masterGua: value.masterGua,
      stoveGua: value.stoveGua,
    }
  }

  function saveFengshuiRecord() {
    const input = fengshuiInput.value
    const profile = selectedProfile.value
    if (!fengshuiResult.value || !input || !profile) return
    void saveBazhaiRecordAndAnnounce(
      t('records.titleBazhai', { nickname: personLabel.value }),
      {
        profile_id: profile.id,
        layout: { ...input },
      },
    )
  }

  /** Explicit saves report both outcomes; auto-saves report failures only. */
  async function saveRecordAndAnnounce(
    save: (title: string, input: unknown, output?: unknown) => Promise<boolean>,
    title: string,
    input: unknown,
    output?: unknown,
  ) {
    const saved = await save(title, input, output)
    if (saved) options.onNotice?.(t('flow.recordSaved'))
    else options.onError(t('records.saveFailed'))
  }

  const saveBondRecordAndAnnounce = (title: string, input: unknown) =>
    saveRecordAndAnnounce(saveBondRecord, title, input)
  const saveBazhaiRecordAndAnnounce = (title: string, input: unknown) =>
    saveRecordAndAnnounce(saveBazhaiRecord, title, input)

  function computeNaming() {
    return namingTask.run(async () => {
      namingResource.reset()
      namingEngineInput.value = profileBirthInput()
    }, t('flow.namingFailed'))
  }

  const castingLoading = castingTask.loading

  function fetchCastingChart(): Promise<LiuyaoChartResult> {
    const receipt = casting.value
    const location = castingLocation.value
    if (!receipt || !location) {
      return Promise.reject(new Error('Casting receipt and location are required'))
    }
    return divinationChartFor({
      standard_time: standardTimeFromInstant(castingInstant.value, location.timezone),
      longitude: location.longitude,
      casting: receipt,
      yongShen: '世爻',
    })
  }

  /** Retry the chart for the current receipt; never draws a new hexagram. */
  async function retryDivinationChart() {
    castingError.value = ''
    try {
      castingChart.value = await fetchCastingChart()
    } catch (error) {
      castingChart.value = null
      castingError.value = toUserMessage(error, t('services.divination.failed'))
    }
  }

  function castDivination(value: DivinationCastingFormValue) {
    // Casting is a mutation (each call draws a new receipt); its failures render
    // inside the focused form. Only the chart query is a resource lifecycle.
    return castingTask.run(async () => {
      castingError.value = ''
      castingChart.value = null
      castingQuestion.value = value.question.trim()

      const location = value.location
      if (!location) {
        castingError.value = t('services.divination.placeRequired')
        return
      }

      const castingTimestamp = new Date().toISOString()

      try {
        const { casting: nextCasting } = await divinationCastingFor({
          mode: value.castingMode,
        })

        casting.value = nextCasting
        castingInstant.value = castingTimestamp
        castingLocation.value = location
        const nextChart = await fetchCastingChart()
        castingChart.value = nextChart

        void saveLiuyaoRecord(
          t('records.titleLiuyao', { question: castingQuestion.value.slice(0, 30) }),
          {
            question: castingQuestion.value,
            casting_mode: value.castingMode,
            casting_instant: castingTimestamp,
            location,
          },
          nextChart as unknown as Record<string, unknown>,
        ).then(saved => {
          if (!saved) options.onError(t('records.saveFailed'))
        })
      } catch (error) {
        castingError.value = toUserMessage(error, t('services.divination.failed'))
      }
    }, t('services.divination.failed'))
  }

  /** Restore a marketing casting receipt without drawing a new hexagram. */
  async function hydrateDivination(context: ConsultationContext) {
    const receipt = context.casting
    const location = context.location
    if (!receipt || !location) return

    castingError.value = ''
    castingQuestion.value = context.question ?? ''
    castingInstant.value = context.casting_instant ?? new Date().toISOString()
    castingLocation.value = {
      place: '',
      longitude: location.longitude,
      timezone: location.timezone,
      source: 'saved',
    }
    casting.value = receipt
    castingChart.value = null
    await retryDivinationChart()
  }

  /** Channel changes must not leak deterministic artifacts between people or domains. */
  function resetResults() {
    casting.value = null
    castingInstant.value = ''
    castingLocation.value = null
    castingChart.value = null
    castingError.value = ''
    hepanInput.value = null
    hepanEngineInput.value = null
    hepanResource.reset()
    fengshuiInput.value = null
    fengshuiEngineInput.value = null
    fengshuiResource.reset()
    namingEngineInput.value = null
    namingResource.reset()
  }

  return {
    // Divination
    castingLoading,
    castingError,
    castingQuestion,
    casting,
    castingInstant,
    castingLocation,
    castingChart,
    castDivination,
    hydrateDivination,
    retryDivinationChart,

    // Hepan
    hepanLoading,
    hepanError,
    hepanInput,
    hepanResult,
    computeHepan,
    reloadHepan: hepanResource.reload,
    saveHepanRecord,

    // Fengshui
    fengshuiLoading,
    fengshuiError,
    fengshuiInput,
    fengshuiResult,
    fengshuiBirthYear,
    fengshuiGender,
    computeFengshui,
    reloadFengshui: fengshuiResource.reload,
    saveFengshuiRecord,

    // Naming
    namingLoading,
    namingError,
    namingResult,
    computeNaming,
    reloadNaming: namingResource.reload,
    profileConsultationBirth,

    resetResults,
  }
}
