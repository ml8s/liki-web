import { computed, ref, type MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BirthProfile } from '@/api/birth-profiles'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'
import type { FlowHepanFormValue } from '@/components/flow/forms/FlowHepanForm.vue'
import { birthInputFromFields } from '@shared/components/form/birth-fields'
import type { FengshuiLayoutValue } from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import type { ArtifactPayloadByKind } from '@/components/flow/artifacts/types'
import { useToolRecords } from '@/composables/useToolRecords'
import { chartInputFromBirth } from '@shared/domain/birth'
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
import { toUserMessage } from '@/utils/errors'

export interface ConsultationToolOptions {
  selectedProfile: MaybeRefOrGetter<BirthProfile | null>
  personLabel: MaybeRefOrGetter<string>
  /** Tool adapters own technical messages; the view decides where to show them. */
  onError: (message: string) => void
  /** Transient positive confirmations (e.g. record saved); the view renders
   *  them through a polite live region. */
  onNotice?: (message: string) => void
}

/**
 * Owns focused-tool state and deterministic artifact computation. It never
 * creates chat sessions: follow-up routing stays with the consultation view.
 */
export function useConsultationTools(options: ConsultationToolOptions) {
  const { t } = useI18n()
  const selectedProfile = computed(() => toValue(options.selectedProfile))
  const personLabel = computed(() => toValue(options.personLabel))

  const castingLoading = ref(false)
  const castingError = ref('')
  const castingQuestion = ref('')
  const casting = ref<LiuyaoCasting | null>(null)
  const castingInstant = ref('')
  const castingLocation = ref<import('@shared/components/divination/types').DivinationCastingFormValue['location']>(null)
  const castingChart = ref<LiuyaoChartResult | null>(null)

  const hepanLoading = ref(false)
  const hepanInput = ref<FlowHepanFormValue | null>(null)
  const hepanResult = ref<ArtifactPayloadByKind['hepan'] | null>(null)

  const fengshuiLoading = ref(false)
  const fengshuiInput = ref<FengshuiLayoutValue | null>(null)
  const fengshuiResult = ref<ArtifactPayloadByKind['fengshui'] | null>(null)

  const namingLoading = ref(false)
  const namingResult = ref<ArtifactPayloadByKind['naming'] | null>(null)

  const {
    save: saveLiuyaoRecord,
  } = useToolRecords('liuyao')
  const {
    save: saveBondRecord,
  } = useToolRecords('bond')
  const {
    save: saveBazhaiRecord,
  } = useToolRecords('bazhai')

  const fengshuiBirthYear = computed(() => {
    const date = selectedProfile.value?.birth.date ?? ''
    const year = Number.parseInt(date.slice(0, 4), 10)
    return Number.isFinite(year) && year >= 1900 && year <= 2100 ? year : null
  })

  const fengshuiGender = computed(() => {
    const gender = selectedProfile.value?.birth.gender
    return gender === 'female' || gender === 'male' ? gender : ''
  })

  function profileChartInput() {
    const profile = selectedProfile.value
    if (!profile?.birth.instant || !profile.birth.location.timezone || profile.birth.location.longitude == null) {
      throw new TypeError('A complete primary birth profile is required')
    }
    return {
      instant: profile.birth.instant,
      timezone: profile.birth.location.timezone,
      longitude: profile.birth.location.longitude,
      gender: profile.birth.gender,
    }
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

  async function computeHepan(value: FlowHepanFormValue) {
    if (hepanLoading.value) return
    hepanLoading.value = true
    hepanResult.value = null
    try {
      const otherBirth = birthInputFromFields(value.secondBirth)
      const otherInput = await chartInputFromBirth(otherBirth)
      const hepan = await hepanForChartInputs(profileChartInput(), {
        ...otherInput,
        gender: otherBirth.gender,
      })
      hepanInput.value = value
      hepanResult.value = {
        chartA: hepan.chartA,
        chartB: hepan.chartB,
        bond: hepan.bond,
        nameA: personLabel.value,
      }
    } catch (error) {
      options.onError(toUserMessage(error, t('services.hepan.failed')))
    } finally {
      hepanLoading.value = false
    }
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

  async function computeFengshui(value: FengshuiLayoutValue) {
    if (fengshuiLoading.value) return
    const profile = selectedProfile.value
    const birthYear = fengshuiBirthYear.value
    if (!profile || birthYear === null || !profile.birth.date) {
      options.onError(t('services.fengshui.invalidBirthYear'))
      return
    }

    fengshuiLoading.value = true
    fengshuiResult.value = null
    try {
      const [bazhai, xuankong] = await Promise.all([
        bazhaiChartFor(birthYear, profile.birth.gender),
        xuankongChartFor(value.periodDate, value.zuoShan),
      ])
      const layout = await bazhaiLayoutFor(
        bazhai.ming_gua.gua.name,
        value.doorGua,
        value.masterGua,
        value.stoveGua,
      )
      fengshuiInput.value = value
      fengshuiResult.value = {
        bazhai,
        layout,
        xuankong,
      }
    } catch (error) {
      options.onError(toUserMessage(error, t('services.fengshui.failed')))
    } finally {
      fengshuiLoading.value = false
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

  async function computeNaming() {
    if (namingLoading.value) return
    namingLoading.value = true
    namingResult.value = null
    try {
      const chart = await baziChartFor(profileBirthInput())
      namingResult.value = { assessment: await namingAssessmentFor(chart) }
    } catch (error) {
      options.onError(toUserMessage(error, t('flow.namingFailed')))
    } finally {
      namingLoading.value = false
    }
  }

  async function castDivination(value: DivinationCastingFormValue) {
    if (castingLoading.value) return

    castingLoading.value = true
    castingError.value = ''
    castingChart.value = null
    castingQuestion.value = value.question.trim()

    const location = value.location
    if (!location) {
      castingLoading.value = false
      castingError.value = t('services.divination.placeRequired')
      return
    }

    const castingTimestamp = new Date().toISOString()

    try {
      const { casting: nextCasting } = await divinationCastingFor({
        mode: value.castingMode,
      })
      const nextChart = await divinationChartFor({
        instant: castingTimestamp,
        timezone: location.timezone,
        longitude: location.longitude,
        casting: nextCasting,
        yongShen: '世爻',
      })

      casting.value = nextCasting
      castingInstant.value = castingTimestamp
      castingLocation.value = location
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
    } finally {
      castingLoading.value = false
    }
  }

  /** Channel changes must not leak deterministic artifacts between people or domains. */
  function resetResults() {
    casting.value = null
    castingInstant.value = ''
    castingLocation.value = null
    castingChart.value = null
    castingError.value = ''
    hepanInput.value = null
    hepanResult.value = null
    fengshuiInput.value = null
    fengshuiResult.value = null
    namingResult.value = null
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

    // Hepan
    hepanLoading,
    hepanInput,
    hepanResult,
    computeHepan,
    saveHepanRecord,

    // Fengshui
    fengshuiLoading,
    fengshuiInput,
    fengshuiResult,
    fengshuiBirthYear,
    fengshuiGender,
    computeFengshui,
    saveFengshuiRecord,

    // Naming
    namingLoading,
    namingResult,
    computeNaming,
    profileConsultationBirth,

    resetResults,
  }
}
