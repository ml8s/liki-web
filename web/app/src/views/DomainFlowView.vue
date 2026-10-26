<template>
  <AppFlowShell
    ref="shellRef"
    :person-label="personLabel"
    :domain-label="domainLabel"
    :restore-visible="restoreVisible"
    @restore="restoreLatest"
  >
    <div
      ref="documentBoundary"
      class="artifact-layer"
      :class="{ 'artifact-layer--sidecar': selectedArtifact && artifactOpen }"
    >
      <div class="artifact-layer__document">
        <el-alert
          v-if="chat.error"
          :title="chat.error"
          type="error"
          :closable="false"
          show-icon
          class="flow-error"
        />
        <el-alert
          v-if="consultationError"
          :title="consultationError"
          type="error"
          :closable="false"
          show-icon
          class="flow-error"
        />

        <ConsultationSpine
          :person-label="personLabel"
          :domain-label="domainLabel"
          :loading="initialLoading"
          :error="profileError"
          :has-profile="hasVisibleProfile"
          @create-profile="openCreateProfile"
          @retry-profiles="reloadProfiles"
        >
          <ChatStreamStatus :phase="chat.phase" />
          <FlowHepanForm
            v-if="showHepanForm"
            :person-label="personLabel"
            :loading="hepanLoading"
            @submit="computeHepan"
          />
          <div
            v-if="hepanArtifact"
            class="flow-inline-actions"
          >
            <el-button
              size="small"
              @click="saveHepanRecord"
            >
              {{ $t('records.save') }}
            </el-button>
          </div>
          <template v-if="showDivinationCasting">
            <el-alert
              v-if="castingError"
              :title="castingError"
              type="error"
              :closable="false"
              show-icon
              class="flow-error"
            />
            <FlowDivinationForm
              ref="castingFormRef"
              :loading="castingLoading"
              @submit="castDivination"
            />
          </template>
          <template v-else>
            <ChatStreamStatus :phase="chat.phase" />
            <ConsultationDocument
              :messages="chat.messages"
              :loading="chat.loadingMessages"
              :streaming="chat.sending"
            />
          </template>
        </ConsultationSpine>

        <div
          v-if="artifacts.length"
          ref="artifactDockRef"
          class="artifact-dock"
        >
          <ArtifactCapsule
            v-for="artifact in artifacts"
            :key="artifact.id"
            :descriptor="artifact"
            :open="artifactOpen && artifactSelectedId === artifact.id"
            :surface-id="`artifact-${artifact.id}`"
            @toggle="toggleArtifact(artifact.id)"
          />
        </div>
      </div>

      <ArtifactSurface
        v-if="selectedArtifact"
        :descriptor="selectedArtifact"
        :open="artifactOpen"
        :mode="desktopMode"
        @close="closeArtifactAndRestoreFocus"
      />
    </div>

    <template #actions>
      <FlowFollowButton
        v-if="followVisible"
        :label="$t('flow.newContent')"
        @follow="followLatest"
      />
      <ParameterComposer
        v-model="question"
        v-model:domain-id="flow.domainId"
        :domains="appDomains"
        :person-label="personLabel"
        :domain-label="domainLabel"
        :sending="chat.sending"
        :show-domain-strip="true"
        @select-person="personSheetOpen = true"
        @select-domain="domainSheetOpen = true"
        @submit="submitQuestion"
        @stop="chat.stopGeneration()"
      />
      <AppExperienceSwitcher
        target="classic"
        variant="dock"
      >
        {{ $t('nav.classic') }}
      </AppExperienceSwitcher>
    </template>
  </AppFlowShell>

  <PersonSwitchSheet
    v-model="personSheetOpen"
    :profiles="profiles.list"
    :selected-person-id="flow.personId"
    :loading="profiles.loading"
    :error="profiles.error"
    @select="selectPerson"
    @create="openCreateProfile"
    @manage="openManageProfiles"
    @retry="reloadProfiles"
  />

  <DomainChooserSheet
    v-model="domainSheetOpen"
    :domains="appDomains"
    :selected-domain-id="flow.domainId"
    @select="selectDomain"
    @new-consultation="startNewConsultation"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppFlowShell from '@/components/flow/AppFlowShell.vue'
import AppExperienceSwitcher from '@/components/app/AppExperienceSwitcher.vue'
import FlowFollowButton from '@/components/flow/FlowFollowButton.vue'
import ConsultationSpine from '@/components/flow/ConsultationSpine.vue'
import ParameterComposer from '@/components/flow/ParameterComposer.vue'
import PersonSwitchSheet from '@/components/flow/PersonSwitchSheet.vue'
import DomainChooserSheet from '@/components/flow/DomainChooserSheet.vue'
import ConsultationDocument from '@/components/flow/ConsultationDocument.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import FlowDivinationForm from '@/components/flow/forms/FlowDivinationForm.vue'
import FlowHepanForm, { type FlowHepanFormValue } from '@/components/flow/forms/FlowHepanForm.vue'
import ArtifactCapsule from '@/components/flow/artifacts/ArtifactCapsule.vue'
import ArtifactSurface from '@/components/flow/artifacts/ArtifactSurface.vue'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useChatStore } from '@/stores/chat'
import { useDomainFlowStore } from '@/stores/domain-flow'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useDomainArtifacts } from '@/composables/useDomainArtifacts'
import { useToolRecords } from '@/composables/useToolRecords'
import {
  divinationCastingFor,
  divinationChartFor,
  type LiuyaoCasting,
  type LiuyaoChartResult,
} from '@shared/domain/divination'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'
import type { ConsultationContext } from '@shared/domain/consultation'
import { birthInputFromFields } from '@shared/components/form/birth-fields'
import { chartInputFromBirth } from '@shared/domain/birth'
import { hepanForChartInputs } from '@shared/domain/hepan'
import { consultationBirthFromFields } from '@shared/domain/consultation'
import type { ArtifactPayloadByKind } from '@/components/flow/artifacts/types'
import {
  normalizeAppDomain,
  appDomains,
  appDomainById,
  bridgeLegacyIntent,
  domainForSession,
  normalizeLegacyScenario,
  type AppDomainId,
  type LegacyIntentBridgeResult,
} from '@shared/domain/flow'
import { firstQueryValue } from '@/utils/query'
import { toUserMessage } from '@/utils/errors'
import { useNearBottom } from '@/composables/useNearBottom'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const profiles = useBirthProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()
const flow = useDomainFlowStore()

const personSheetOpen = ref(false)
const domainSheetOpen = ref(false)
const consultationError = ref('')
const documentBoundary = ref<HTMLElement | null>(null)
const shellRef = ref<InstanceType<typeof AppFlowShell> | null>(null)
const { nearBottom, measure: measureNearBottom } = useNearBottom(
  () => shellRef.value?.getMainElement() ?? null,
)
const followVisible = ref(false)
const artifactDockRef = ref<HTMLElement | null>(null)
const castingFormRef = ref<InstanceType<typeof FlowDivinationForm> | null>(null)
const castingLoading = ref(false)
const castingError = ref('')
const castingQuestion = ref('')
const casting = ref<LiuyaoCasting | null>(null)
const castingInstant = ref('')
const castingLocation = ref<DivinationCastingFormValue['location']>(null)
const castingChart = ref<LiuyaoChartResult | null>(null)
const hepanLoading = ref(false)
const hepanInput = ref<FlowHepanFormValue | null>(null)
const hepanResult = ref<ArtifactPayloadByKind['hepan'] | null>(null)
let mountToken = 0
let artifactFocusToken = 0

const question = computed({
  get: () => flow.getDraft(),
  set: value => flow.setDraft(value),
})

const selectedProfile = computed(() =>
  profiles.list.find(profile => profile.id === flow.personId) ?? null,
)

const personLabel = computed(() => selectedProfile.value?.nickname ?? '')
const domainLabel = computed(() => t(flow.currentDomain.labelKey))
const profilePolicy = computed(() => flow.currentDomain.profilePolicy)
const profileRequired = computed(() => profilePolicy.value !== 'optional')
const hasVisibleProfile = computed(() =>
  profilePolicy.value === 'optional'
    ? true
    : Boolean(selectedProfile.value),
)

const visibleSessions = computed(() =>
  chat.sessions
    .filter(session => (session.profile_id || null) === flow.personId)
    .filter(session => domainForSession(session) === flow.domainId)
    .sort((left, right) => right.updated_at.localeCompare(left.updated_at)),
)

const latestVisibleSession = computed(() => visibleSessions.value[0] ?? null)

const restoreVisible = computed(() =>
  Boolean(latestVisibleSession.value) &&
  chat.activeSession?.id !== latestVisibleSession.value?.id &&
  !chat.loadingMessages,
)

const initialLoading = computed(() =>
  profiles.loading || entitlements.loading || chat.loadingSessions,
)

const profileError = computed(() => profiles.error || entitlements.error)
const showDivinationCasting = computed(() => flow.domainId === 'divination' && !castingChart.value)
const showHepanForm = computed(() => flow.domainId === 'natal' && routingSeed.value === 'hepan')

const {
  save: saveToolRecord,
} = useToolRecords('liuyao')
const {
  save: saveBondRecord,
} = useToolRecords('bond')

const artifactContext = computed(() => {
  const session = chat.activeSession
  if (!session?.context) return null
  if ((session.profile_id || null) !== flow.personId) return null
  if (domainForSession(session) !== flow.domainId) return null
  return session.context
})

const {
  artifacts,
  open: artifactOpen,
  selectedArtifact,
  selectedId: artifactSelectedId,
  desktopMode,
  closeArtifact,
  toggleArtifact,
} = useDomainArtifacts({
  channelKey: computed(() => `${flow.personId ?? 'none'}:${flow.domainId}`),
  domainId: computed(() => flow.domainId),
  context: artifactContext,
  casting,
  chart: castingChart,
  hepan: hepanResult,
})

const routingSeed = computed(() => {
  const value = firstQueryValue(route.query.seed) || firstQueryValue(route.query.service)
  return normalizeLegacyScenario(value)
})

const hepanArtifact = computed(() =>
  artifacts.value.find(artifact => artifact.kind === 'hepan' && artifact.state === 'ready') ?? null,
)

function selectPerson(personId: string) {
  hepanInput.value = null
  hepanResult.value = null
  flow.setPerson(personId)
  personSheetOpen.value = false
}

function selectDomain(domainId: AppDomainId) {
  hepanInput.value = null
  hepanResult.value = null
  flow.setDomain(domainId)
  domainSheetOpen.value = false
}

function openCreateProfile() {
  personSheetOpen.value = false
  void router.push({ name: 'birth-profile-new' })
}

function openManageProfiles() {
  personSheetOpen.value = false
  void router.push({ name: 'birth-profiles' })
}

function startNewConsultation() {
  chat.stopGeneration()
  chat.resetActive()
  flow.clearDraft()
  consultationError.value = ''
  domainSheetOpen.value = false
}

async function restoreLatest() {
  const latest = latestVisibleSession.value
  if (!latest) {
    chat.resetActive()
    return
  }
  await chat.openSession(latest.id)
}

function projectBridgeFailure(result: Extract<LegacyIntentBridgeResult, { status: 'cross-domain' | 'pending-backend' | 'invalid' }>) {
  if (result.status === 'cross-domain') {
    const label = t(appDomainById(result.suggestedDomainId).labelKey)
    consultationError.value = t('flow.crossDomain', { domain: label })
    return
  }
  if (result.status === 'pending-backend') {
    consultationError.value = t('flow.huangliPending')
    return
  }
  consultationError.value = t('errors.requestFailed')
}

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

async function computeHepan(value: FlowHepanFormValue) {
  if (hepanLoading.value) return
  hepanLoading.value = true
  consultationError.value = ''
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
    consultationError.value = toUserMessage(error, t('services.hepan.failed'))
  } finally {
    hepanLoading.value = false
  }
}

function saveHepanRecord() {
  if (!hepanResult.value || !hepanInput.value || !selectedProfile.value) return
  void saveBondRecord(
    t('records.titleBond', { a: personLabel.value, b: hepanInput.value.secondBirth.date }),
    {
      profile_id: selectedProfile.value.id,
      b: { ...hepanInput.value.secondBirth },
      third_party_consent: true,
    },
  )
}

function sameConsultationBirth(
  left: ConsultationContext['second_birth'],
  right: ConsultationContext['second_birth'],
) {
  if (!left || !right) return false
  return left.gender === right.gender
    && left.date === right.date
    && left.time === right.time
    && left.location.longitude === right.location.longitude
    && left.location.timezone === right.location.timezone
}

async function createAndSendWithHepanContext(content: string) {
  const input = hepanInput.value
  const profile = selectedProfile.value
  if (!input || !profile) return
  const context: ConsultationContext = {
    version: 1,
    scenario: 'hepan',
    second_birth: consultationBirthFromFields(input.secondBirth),
    third_party_consent: true,
  }

  try {
    const session = await chat.createSession(profile.id, 'hepan', context)
    chat.activeSession = session
    chat.messages = []
    chat.messagesCursor = ''
    await sendUsingActiveSession(content)
  } catch (error) {
    consultationError.value = toUserMessage(error, t('chat.createFailed'))
  }
}

async function submitQuestion() {
  const content = question.value.trim()
  if (!content || chat.sending) return

  consultationError.value = ''

  const useDivinationContext = flow.domainId === 'divination'
    && Boolean(castingChart.value)
    && Boolean(casting.value)

  if (useDivinationContext) {
    const matchingSession = chat.sessions
      .filter(session => (session.profile_id || null) === flow.personId)
      .find(session => session.context?.casting?.casting_id === casting.value?.casting_id)

    if (matchingSession) {
      if (chat.activeSession?.id !== matchingSession.id) await chat.openSession(matchingSession.id)
      await sendUsingActiveSession(content)
      return
    }

    await createAndSendWithDivinationContext(content)
    return
  }

  if (profileRequired.value && !selectedProfile.value) {
    personSheetOpen.value = true
    return
  }

  if (flow.domainId === 'natal' && routingSeed.value === 'hepan' && (!hepanResult.value || !hepanInput.value)) {
    consultationError.value = t('flow.hepanRequired')
    return
  }

  if (flow.domainId === 'natal' && hepanResult.value && hepanInput.value) {
    const expectedBirth = consultationBirthFromFields(hepanInput.value.secondBirth)
    const matchingSession = chat.sessions
      .filter(session => (session.profile_id || null) === flow.personId)
      .find(session => session.context?.scenario === 'hepan'
        && sameConsultationBirth(session.context.second_birth, expectedBirth))

    if (matchingSession) {
      if (chat.activeSession?.id !== matchingSession.id) await chat.openSession(matchingSession.id)
      await sendUsingActiveSession(content)
      return
    }

    await createAndSendWithHepanContext(content)
    return
  }

  const existing = latestVisibleSession.value
  if (existing) {
    if (chat.activeSession?.id !== existing.id) await chat.openSession(existing.id)
    await sendUsingActiveSession(content)
    return
  }

  const bridgeResult = bridgeLegacyIntent({
    domainId: flow.domainId,
    personId: flow.personId ?? undefined,
    seed: routingSeed.value,
  })

  if (bridgeResult.status !== 'ok') {
    projectBridgeFailure(bridgeResult)
    return
  }

  try {
    const session = await chat.createSession(
      flow.personId ?? '',
      bridgeResult.skill,
    )
    chat.activeSession = session
    chat.messages = []
    chat.messagesCursor = ''
    await sendUsingActiveSession(content)
  } catch (error) {
    consultationError.value = toUserMessage(error, t('chat.createFailed'))
  }
}

async function createAndSendWithDivinationContext(content: string) {
  if (!casting.value || !castingChart.value) return

  const context: ConsultationContext = {
    version: 1,
    scenario: 'divination',
    question: castingQuestion.value,
    casting: casting.value,
    casting_instant: castingInstant.value,
    location: castingLocation.value
      ? {
          longitude: castingLocation.value.longitude,
          timezone: castingLocation.value.timezone,
        }
      : undefined,
  }

  try {
    const session = await chat.createSession(
      flow.personId ?? '',
      'divination',
      context,
    )
    chat.activeSession = session
    chat.messages = []
    chat.messagesCursor = ''
    await sendUsingActiveSession(content)
  } catch (error) {
    consultationError.value = toUserMessage(error, t('chat.createFailed'))
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

    void saveToolRecord(
      t('records.titleLiuyao', { question: castingQuestion.value.slice(0, 30) }),
      {
        question: castingQuestion.value,
        casting_mode: value.castingMode,
        casting_instant: castingTimestamp,
        location,
      },
      nextChart as unknown as Record<string, unknown>,
    )
  } catch (error) {
    castingError.value = toUserMessage(error, t('services.divination.failed'))
  } finally {
    castingLoading.value = false
  }
}

async function sendUsingActiveSession(content: string) {
  if (!chat.activeSession) return
  const question = content
  flow.setDraft('')
  await chat.send(question)
  if (chat.error) flow.setDraft(question)
  await scrollToLatest()
}

async function scrollToLatest() {
  await nextTick()
  measureNearBottom()
  if (!nearBottom.value) {
    followVisible.value = true
    return
  }
  await shellRef.value?.scrollToBottom()
  followVisible.value = false
}

async function followLatest() {
  followVisible.value = false
  await shellRef.value?.scrollToBottom({ focusMain: true })
}

function closeArtifactAndRestoreFocus() {
  const artifactId = artifactSelectedId.value
  closeArtifact()
  const token = ++artifactFocusToken
  window.setTimeout(() => {
    if (token !== artifactFocusToken) return
    const trigger = [...(artifactDockRef.value?.querySelectorAll<HTMLButtonElement>('button') ?? [])]
      .find(button => button.getAttribute('aria-controls') === `artifact-${artifactId}`)
    trigger?.focus({
      preventScroll: true,
    })
  }, 0)
}

onBeforeUnmount(() => {
  artifactFocusToken += 1
})
async function reloadProfiles() {
  await profiles.loadList('', true)
}

watch([() => flow.personId, () => flow.domainId], () => {
  hepanInput.value = null
  hepanResult.value = null
  consultationError.value = ''
  void restoreLatest()
})

watch(nearBottom, (value) => {
  followVisible.value = !value && chat.messages.length > 0
})

watch(() => `${chat.messages.length}:${chat.messages.at(-1)?.content.length ?? 0}`, () => {
  if (nearBottom.value) shellRef.value?.scrollToBottom()
  else followVisible.value = true
})

onMounted(async () => {
  const token = ++mountToken
  const requestedDomain = normalizeAppDomain(firstQueryValue(route.query.domain))
  if (requestedDomain) flow.setDomain(requestedDomain)
  await Promise.allSettled([
    profiles.loadList(),
    entitlements.load(),
    chat.loadSessions(true),
  ])
  if (token !== mountToken) return

  if (!flow.personId) flow.setPerson(profiles.list[0]?.id ?? null)
  await restoreLatest()
})
</script>

<style scoped>
.artifact-layer {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
  gap: var(--space-4);
}

.artifact-layer__document {
  min-width: 0;
  display: grid;
  gap: var(--space-4);
}

.artifact-dock {
  display: grid;
  gap: var(--space-2);
}

@media (min-width: 1024px) {
  .artifact-layer--sidecar {
    grid-template-columns: minmax(0, 1fr) 320px;
  }
}
</style>
