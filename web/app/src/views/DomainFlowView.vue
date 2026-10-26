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
        <AppStatusAlert
          v-if="chat.error"
          class="flow-error"
          type="error"
          :title="chat.error"
        />
        <AppStatusAlert
          v-if="consultationError"
          class="flow-error"
          type="error"
          :title="consultationError"
        />
        <AppStatusAlert
          v-if="sales.paymentMessage"
          class="flow-error"
          :type="sales.paymentState === 'confirmed'
            ? 'success'
            : sales.paymentState === 'unpaid' ? 'warning' : 'info'"
          :title="sales.paymentMessage"
        />
        <AppStatusAlert
          v-if="sales.actionError"
          class="flow-error"
          type="error"
          :title="sales.actionError"
        />
        <AppStatusAlert
          v-if="sales.productsError"
          class="flow-error"
          type="error"
          :title="sales.productsError"
        >
          <el-button
            size="small"
            @click="sales.loadProducts"
          >
            {{ $t('common.retry') }}
          </el-button>
        </AppStatusAlert>

        <ProductPurchaseCard
          v-if="sales.chatProduct && !sales.chatProduct.purchased"
          :product="sales.chatProduct"
          :buying="sales.buying"
          @purchase="purchaseConsultation"
        />

        <ConsultationSpine
          :person-label="personLabel"
          :domain-label="domainLabel"
          :loading="initialLoading"
          :error="profileError"
          :failure-kind="workspaceFailure ?? undefined"
          :has-profile="hasVisibleProfile"
          @create-profile="openCreateProfile"
          @retry-profiles="retryWorkspace"
          @open-account="openManageAccount"
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
          <FlowFengshuiForm
            v-if="showFengshuiForm"
            :person-label="personLabel"
            :birth-year="fengshuiBirthYear"
            :gender="fengshuiGender"
            :loading="fengshuiLoading"
            @submit="computeFengshui"
          />
          <div
            v-if="fengshuiArtifact"
            class="flow-inline-actions"
          >
            <el-button
              size="small"
              @click="saveFengshuiRecord"
            >
              {{ $t('records.save') }}
            </el-button>
          </div>
          <FlowNamingForm
            v-if="showNamingForm"
            :person-label="personLabel"
            :birth-date="selectedProfile?.birth.date ?? ''"
            :birth-time="selectedProfile?.birth.time ?? ''"
            :loading="namingLoading"
            :disabled="!namingBirthReady"
            @submit="computeNaming"
          />
          <template v-if="showDivinationCasting">
            <AppStatusAlert
              v-if="castingError"
              class="flow-error"
              type="error"
              :title="castingError"
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

        <FlowHistoryPanel
          class="flow-history"
          :sessions="visibleSessions"
          :active-session-id="chat.activeSession?.id"
          :loading="chat.loadingSessions"
          :error="chat.error"
          @open="openHistorySession"
          @rename="renameHistorySession"
          @remove="removeHistorySession"
        />
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
      <div class="flow-dock-links">
        <AppExperienceSwitcher
          target="me"
          variant="dock"
        >
          {{ $t('nav.me') }}
        </AppExperienceSwitcher>
        <AppExperienceSwitcher
          target="classic"
          variant="dock"
        >
          {{ $t('nav.classic') }}
        </AppExperienceSwitcher>
      </div>
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
import AppStatusAlert from '@/components/app/AppStatusAlert.vue'
import FlowFollowButton from '@/components/flow/FlowFollowButton.vue'
import ConsultationSpine from '@/components/flow/ConsultationSpine.vue'
import ParameterComposer from '@/components/flow/ParameterComposer.vue'
import PersonSwitchSheet from '@/components/flow/PersonSwitchSheet.vue'
import DomainChooserSheet from '@/components/flow/DomainChooserSheet.vue'
import ConsultationDocument from '@/components/flow/ConsultationDocument.vue'
import FlowHistoryPanel from '@/components/flow/FlowHistoryPanel.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import FlowDivinationForm from '@/components/flow/forms/FlowDivinationForm.vue'
import FlowHepanForm from '@/components/flow/forms/FlowHepanForm.vue'
import FlowFengshuiForm from '@/components/flow/forms/FlowFengshuiForm.vue'
import FlowNamingForm from '@/components/flow/forms/FlowNamingForm.vue'
import ArtifactCapsule from '@/components/flow/artifacts/ArtifactCapsule.vue'
import ArtifactSurface from '@/components/flow/artifacts/ArtifactSurface.vue'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useChatStore } from '@/stores/chat'
import { useDomainFlowStore } from '@/stores/domain-flow'
import type { ChatSession } from '@/api/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useDomainArtifacts } from '@/composables/useDomainArtifacts'
import { useSalesStore } from '@/stores/sales-store'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import { confirmDeleteEntity, promptRenameEntity } from '@/utils/dialogs'
import type { ConsultationContext } from '@shared/domain/consultation'
import { consultationBirthFromFields } from '@shared/domain/consultation'
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
import { useConsultationTools } from '@/composables/useConsultationTools'
import { useOnline } from '@vueuse/core'
import { classifyResourceFailure, type ResourceFailureKind } from '@/utils/resourceStatus'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const profiles = useBirthProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()
const flow = useDomainFlowStore()
const sales = useSalesStore()
const isOnline = useOnline()

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
const {
  castingLoading,
  castingError,
  casting,
  castingQuestion,
  castingInstant,
  castingLocation,
  castingChart,
  castDivination,
  hepanLoading,
  hepanInput,
  hepanResult,
  computeHepan,
  saveHepanRecord,
  fengshuiLoading,
  fengshuiInput,
  fengshuiResult,
  fengshuiBirthYear,
  fengshuiGender,
  computeFengshui,
  saveFengshuiRecord,
  namingLoading,
  namingResult,
  computeNaming,
  profileConsultationBirth,
  resetResults,
} = useConsultationTools({
  selectedProfile,
  personLabel,
  onError: message => {
    consultationError.value = message
  },
})
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
const workspaceFailure = computed<ResourceFailureKind | null>(() => {
  if (!isOnline.value) return 'offline'
  const error = profiles.lastError ?? entitlements.lastError ?? chat.loadSessionsError
  return error ? classifyResourceFailure(error) : null
})
const showDivinationCasting = computed(() => flow.domainId === 'divination' && !castingChart.value)
const showHepanForm = computed(() => flow.domainId === 'natal' && routingSeed.value === 'hepan')
const showFengshuiForm = computed(() => flow.domainId === 'fengshui' && routingSeed.value === 'fengshui')
const showNamingForm = computed(() => flow.domainId === 'naming' && routingSeed.value === 'naming')
const namingBirthReady = computed(() => {
  const birth = selectedProfile.value?.birth
  return Boolean(birth?.gender && birth.date && birth.time && birth.location)
})
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
  fengshui: fengshuiResult,
  naming: namingResult,
})

const routingSeed = computed(() => {
  const value = firstQueryValue(route.query.seed) || firstQueryValue(route.query.service)
  return normalizeLegacyScenario(value)
})

const hepanArtifact = computed(() =>
  artifacts.value.find(artifact => artifact.kind === 'hepan' && artifact.state === 'ready') ?? null,
)
const fengshuiArtifact = computed(() =>
  artifacts.value.find(artifact => artifact.kind === 'fengshui' && artifact.state === 'ready') ?? null,
)

function selectPerson(personId: string) {
  resetResults()
  flow.setPerson(personId)
  personSheetOpen.value = false
}

function selectDomain(domainId: AppDomainId) {
  resetResults()
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

function openManageAccount() {
  void router.push({ name: 'account' })
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

async function openHistorySession(session: ChatSession) {
  if (chat.activeSession?.id === session.id) return
  await chat.openSession(session.id)
}

async function renameHistorySession(session: ChatSession) {
  let value: string | null
  try {
    value = await promptRenameEntity(t('chat.sessions.noun'), session.title)
  } catch (error) {
    chat.error = toUserMessage(error, t('flow.historyActionFailed'))
    return
  }
  if (value === null) return
  try {
    await chat.renameSession(session.id, value)
  } catch (error) {
    chat.error = toUserMessage(error, t('flow.historyActionFailed'))
  }
}

async function removeHistorySession(session: ChatSession) {
  const confirmed = await confirmDeleteEntity(
    t('chat.sessions.noun'),
    session.title,
  )
  if (!confirmed) return
  try {
    await chat.removeSession(session.id)
  } catch (error) {
    chat.error = toUserMessage(error, t('flow.historyActionFailed'))
  }
}

function purchaseConsultation(billingMode: 'one_time' | 'subscription') {
  void sales.buy(billingMode, routingSeed.value ?? flow.currentDomain.baseSkill)
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

function sameConsultationBirth(
  left: ConsultationContext['second_birth'] | ConsultationContext['birth'],
  right: ConsultationContext['second_birth'] | ConsultationContext['birth'],
) {
  if (!left || !right) return false
  return left.gender === right.gender
    && left.date === right.date
    && left.time === right.time
    && left.location.longitude === right.location.longitude
    && left.location.timezone === right.location.timezone
}

function sameFengshuiLayout(
  left: ConsultationContext['layout'],
  right: ConsultationContext['layout'],
) {
  if (!left || !right) return false
  return left.zuo_shan === right.zuo_shan
    && left.period_date === right.period_date
    && left.door_gua === right.door_gua
    && left.master_gua === right.master_gua
    && left.stove_gua === right.stove_gua
}

async function openMatchingAndSend(
  matcher: (session: ChatSession) => boolean,
  content: string,
) {
  const matchingSession = chat.sessions
    .filter(session => (session.profile_id || null) === flow.personId)
    .find(matcher)

  if (!matchingSession) return false
  if (chat.activeSession?.id !== matchingSession.id) await chat.openSession(matchingSession.id)
  await sendUsingActiveSession(content)
  return true
}

async function createAndSendWithSession(
  skill: string,
  context: ConsultationContext | undefined,
  content: string,
) {
  try {
    const session = context === undefined
      ? await chat.createSession(flow.personId ?? '', skill)
      : await chat.createSession(flow.personId ?? '', skill, context)
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

  if (flow.domainId === 'divination' && casting.value && castingChart.value) {
    const sent = await openMatchingAndSend(
      session => session.context?.casting?.casting_id === casting.value?.casting_id,
      content,
    )
    if (sent) return
    await createAndSendWithSession('divination', {
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
    }, content)
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

  if (flow.domainId === 'fengshui' && routingSeed.value === 'fengshui' && (!fengshuiResult.value || !fengshuiInput.value)) {
    consultationError.value = t('flow.fengshuiRequired')
    return
  }

  if (flow.domainId === 'naming' && routingSeed.value === 'naming' && !namingResult.value) {
    consultationError.value = t('flow.namingRequired')
    return
  }

  if (flow.domainId === 'naming' && namingResult.value) {
    const context: ConsultationContext = {
      version: 1,
      scenario: 'naming',
      birth: profileConsultationBirth(),
    }
    const sent = await openMatchingAndSend(
      session => session.context?.scenario === 'naming'
        && sameConsultationBirth(session.context.birth, context.birth),
      content,
    )
    if (sent) return
    await createAndSendWithSession('naming', context, content)
    return
  }

  if (flow.domainId === 'fengshui' && fengshuiResult.value && fengshuiInput.value) {
    const context: ConsultationContext = {
      version: 1,
      scenario: 'fengshui',
      layout: {
        zuo_shan: fengshuiInput.value.zuoShan,
        period_date: fengshuiInput.value.periodDate,
        door_gua: fengshuiInput.value.doorGua,
        master_gua: fengshuiInput.value.masterGua,
        stove_gua: fengshuiInput.value.stoveGua,
      },
    }
    const sent = await openMatchingAndSend(
      session => session.context?.scenario === 'fengshui'
        && sameFengshuiLayout(session.context.layout, context.layout),
      content,
    )
    if (sent) return
    await createAndSendWithSession('fengshui', context, content)
    return
  }

  if (flow.domainId === 'natal' && hepanResult.value && hepanInput.value) {
    const context: ConsultationContext = {
      version: 1,
      scenario: 'hepan',
      second_birth: consultationBirthFromFields(hepanInput.value.secondBirth),
      third_party_consent: true,
    }
    const sent = await openMatchingAndSend(
      session => session.context?.scenario === 'hepan'
        && sameConsultationBirth(session.context.second_birth, context.second_birth),
      content,
    )
    if (sent) return
    await createAndSendWithSession('hepan', context, content)
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

  await createAndSendWithSession(bridgeResult.skill, undefined, content)
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

async function retryWorkspace() {
  await Promise.allSettled([
    profiles.loadList('', true),
    entitlements.load(true),
    chat.loadSessions(true),
    sales.loadProducts(),
  ])
}

watch([() => flow.personId, () => flow.domainId], () => {
  resetResults()
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
    sales.loadProducts(),
  ])
  if (token !== mountToken) return

  if (!flow.personId) flow.setPerson(profiles.list[0]?.id ?? null)
  await restoreLatest()

  const paymentResult = await sales.handlePaymentReturn(route.query)
  if (paymentResult.action === 'enter_chat') {
    const sessionId = await sales.enterInterpretation(
      flow.personId ?? undefined,
      paymentResult.scenarioId,
    )
    if (sessionId) {
      await router.replace({
        path: '/',
        query: route.query.locale ? { locale: firstQueryValue(route.query.locale) } : {},
      })
      await chat.openSession(sessionId)
    }
  } else if (paymentResult.action === 'show_message') {
    await router.replace({
      path: '/',
      query: route.query.locale ? { locale: firstQueryValue(route.query.locale) } : {},
    })
  }
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
.flow-history {
  margin-top: var(--space-4);
}
.flow-dock-links {
  display: grid;
  gap: var(--space-2);
}
</style>
