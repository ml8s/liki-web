<template>
  <AppFlowShell
    ref="shellRef"
    :person-label="personLabel"
    :domain-label="domainLabel"
    :domains="appDomains"
    :selected-domain-id="flow.domainId"
    :partner-visible="auth.hasPartnerRole"
    :restore-visible="restoreVisible"
    :sidecar-open="Boolean(selectedArtifact && artifactOpen && !showFocusedForm)"
    :sidecar-mode="desktopMode"
    @domain-select="selectDomain"
    @context-select="handleContextAction"
    @restore="restoreLatest"
  >
    <div
      ref="documentBoundary"
      class="artifact-layer"
    >
      <div class="artifact-layer__document">
        <div
          v-if="primaryError"
          ref="errorFocusRef"
          class="flow-error-focus"
          :tabindex="primaryError ? -1 : undefined"
        >
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
        </div>
        <AppLiveRegion
          v-if="draftNotice"
          class="flow-draft-notice"
          screen-only
        >
          {{ draftNotice }}
        </AppLiveRegion>
        <AppLiveRegion
          v-if="statusNotice"
          class="flow-status-notice"
          screen-only
        >
          {{ statusNotice }}
        </AppLiveRegion>
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
          :profile-policy="profilePolicy"
          @ask-divination="askDivinationWithoutProfile"
          @open-huangli="openHuangliCompatibility"
          @create-profile="openCreateProfile"
          @retry-profiles="retryWorkspace"
          @open-account="openManageAccount"
        >
          <ChatStreamStatus :phase="chat.phase" />
          <DomainSlotForm
            v-if="showHepanForm"
            ref="hepanSlotRef"
            :domain-label="domainLabel"
            :title="$t('services.hepan.title')"
            :description="$t('services.hepan.subtitle')"
            @back="exitFocusedForm"
          >
            <FlowHepanForm
              :person-label="personLabel"
              :loading="hepanLoading"
              @submit="computeHepan"
            />
          </DomainSlotForm>
          <AppStatusAlert
            v-if="hepanError"
            class="flow-error"
            type="error"
            :title="hepanError"
          >
            <el-button
              size="small"
              @click="reloadHepan"
            >
              {{ $t('common.retry') }}
            </el-button>
          </AppStatusAlert>
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
          <DomainSlotForm
            v-if="showFengshuiForm"
            ref="fengshuiSlotRef"
            :domain-label="domainLabel"
            :title="$t('services.fengshui.pageTitle')"
            :description="$t('services.fengshui.subtitle')"
            @back="exitFocusedForm"
          >
            <FlowFengshuiForm
              :person-label="personLabel"
              :birth-year="fengshuiBirthYear"
              :gender="fengshuiGender"
              :loading="fengshuiLoading"
              :initial-layout="fengshuiHandoffLayout"
              @submit="computeFengshui"
            />
          </DomainSlotForm>
          <AppStatusAlert
            v-if="fengshuiError"
            class="flow-error"
            type="error"
            :title="fengshuiError"
          >
            <el-button
              size="small"
              @click="reloadFengshui"
            >
              {{ $t('common.retry') }}
            </el-button>
          </AppStatusAlert>
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
          <DomainSlotForm
            v-if="showNamingForm"
            ref="namingSlotRef"
            :domain-label="domainLabel"
            :title="$t('services.naming.label')"
            :description="$t('services.naming.description')"
            @back="exitFocusedForm"
          >
            <FlowNamingForm
              :person-label="personLabel"
              :birth-date="selectedProfile?.birth.date ?? ''"
              :birth-time="selectedProfile?.birth.time ?? ''"
              :loading="namingLoading"
              :disabled="!namingBirthReady"
              @submit="computeNaming"
            />
          </DomainSlotForm>
          <AppStatusAlert
            v-if="namingError"
            class="flow-error"
            type="error"
            :title="namingError"
          >
            <el-button
              size="small"
              @click="reloadNaming"
            >
              {{ $t('common.retry') }}
            </el-button>
          </AppStatusAlert>
          <template v-if="showDivinationCasting">
            <AppStatusAlert
              v-if="castingError"
              class="flow-error"
              type="error"
              :title="castingError"
            >
              <el-button
                v-if="casting"
                size="small"
                @click="retryDivinationChart"
              >
                {{ $t('common.retry') }}
              </el-button>
            </AppStatusAlert>
            <DomainSlotForm
              ref="divinationSlotRef"
              :domain-label="domainLabel"
              :title="$t('services.divination.pageTitle')"
              :description="$t('services.divination.subtitle')"
              @back="exitFocusedForm"
            >
              <FlowDivinationForm
                ref="castingFormRef"
                :loading="castingLoading"
                @submit="castDivination"
              />
            </DomainSlotForm>
          </template>
          <template v-else>
            <ChatStreamStatus :phase="chat.phase" />
            <ConsultationDocument
              :messages="chat.messages"
              :loading="chat.loadingMessages"
              :streaming="chat.sending"
            />
          </template>

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
        </ConsultationSpine>

        <FlowHistoryPanel
          class="flow-history"
          :sessions="visibleSessions"
          :active-session-id="chat.activeSession?.id"
          :loading="chat.loadingSessions"
          :error="chat.error"
          @open="sessionHistory.openSession"
          @rename="sessionHistory.renameSession"
          @remove="sessionHistory.removeSession"
        />
      </div>
    </div>

    <template #sidecar>
      <ArtifactSurface
        v-if="selectedArtifact"
        :descriptor="selectedArtifact"
        :open="artifactOpen"
        :mode="desktopMode"
        @close="closeArtifactAndRestoreFocus"
        @desktop-mode="setDesktopMode"
      />
    </template>

    <template #actions>
      <FlowFollowButton
        v-if="followVisible && !showFocusedForm"
        :label="$t('flow.newContent')"
        @follow="followLatest"
      />
      <div
        v-show="!showFocusedForm"
        class="flow-composer-holder"
      >
        <ParameterComposer
          v-model="question"
          v-model:domain-id="flow.domainId"
          :domains="appDomains"
          :person-label="personLabel"
          :domain-label="domainLabel"
          :sending="chat.sending || isSubmitting"
          :show-domain-strip="!chat.activeSession"
          @select-person="personSheetOpen = true"
          @select-domain="domainSheetOpen = true"
          @submit="submitQuestion"
          @stop="chat.stopGeneration()"
        />
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
import AppStatusAlert from '@/components/app/AppStatusAlert.vue'
import FlowFollowButton from '@/components/flow/FlowFollowButton.vue'
import ConsultationSpine from '@/components/flow/ConsultationSpine.vue'
import ParameterComposer from '@/components/flow/ParameterComposer.vue'
import PersonSwitchSheet from '@/components/flow/PersonSwitchSheet.vue'
import DomainChooserSheet from '@/components/flow/DomainChooserSheet.vue'
import ConsultationDocument from '@/components/flow/ConsultationDocument.vue'
import FlowHistoryPanel from '@/components/flow/FlowHistoryPanel.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import AppLiveRegion from '@/components/app/AppLiveRegion.vue'
import FlowDivinationForm from '@/components/flow/forms/FlowDivinationForm.vue'
import DomainSlotForm from '@/components/flow/forms/DomainSlotForm.vue'
import FlowHepanForm from '@/components/flow/forms/FlowHepanForm.vue'
import FlowFengshuiForm from '@/components/flow/forms/FlowFengshuiForm.vue'
import type { FengshuiLayoutValue } from '@shared/components/fengshui/FengshuiLayoutFields.vue'
import FlowNamingForm from '@/components/flow/forms/FlowNamingForm.vue'
import ArtifactCapsule from '@/components/flow/artifacts/ArtifactCapsule.vue'
import ArtifactSurface from '@/components/flow/artifacts/ArtifactSurface.vue'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useChatStore } from '@/stores/chat'
import { useDomainFlowStore } from '@/stores/domain-flow'
import { useAuthStore } from '@/stores/auth'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useDomainArtifacts } from '@/composables/useDomainArtifacts'
import { useSalesStore } from '@/stores/sales-store'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import type { ConsultationContext } from '@shared/domain/consultation'
import {
  consultationBirthFromFields,
  sameConsultationBirth,
  sameConsultationLayout,
} from '@shared/domain/consultation'
import {
  normalizeAppDomain,
  appDomains,
  appDomainById,
  bridgeLegacyIntent,
  domainForScenario,
  domainForSession,
  marketingScenarioToDomain,
  normalizeLegacyScenario,
  type AppDomainId,
  type LegacyIntentBridgeResult,
} from '@shared/domain/flow'
import { firstQueryValue } from '@/utils/query'
import { useNearBottom } from '@/composables/useNearBottom'
import { useConsultationTools } from '@/composables/useConsultationTools'
import { useConsultationTelemetry } from '@/composables/useConsultationTelemetry'
import { useConsultationHistory } from '@/composables/useConsultationHistory'
import { useConsultationSubmit } from '@/composables/useConsultationSubmit'
import { useUnloadGuard } from '@/composables/useUnloadGuard'
import { confirmAction } from '@/utils/dialogs'
import { consumeConsultationHandoff } from '@shared/utils/handoff'
import { useOnline } from '@vueuse/core'
import { classifyResourceFailure, type ResourceFailureKind } from '@/utils/resourceStatus'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const profiles = useBirthProfilesStore()
const auth = useAuthStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()
const flow = useDomainFlowStore()
const sales = useSalesStore()
const isOnline = useOnline()
const telemetry = useConsultationTelemetry()

const sessionHistory = useConsultationHistory({
  personId: computed(() => flow.personId),
  domainId: computed(() => flow.domainId),
})

const submit = useConsultationSubmit({
  personId: computed(() => flow.personId),
  onError: message => {
    consultationError.value = message
  },
  onDraftPreserved: () => {
    draftNotice.value = t('flow.draftPreserved')
  },
  onDraftRestored: () => {
    draftNotice.value = t('flow.draftRestored')
  },
  onAfterSend: () => {
    void scrollToLatest()
  },
})
const isSubmitting = submit.isSubmitting

const statusNotice = ref('')
let statusNoticeTimer = 0
function announceStatus(message: string) {
  statusNotice.value = message
  window.clearTimeout(statusNoticeTimer)
  statusNoticeTimer = window.setTimeout(() => {
    statusNotice.value = ''
  }, 4000)
}

const personSheetOpen = ref(false)
const domainSheetOpen = ref(false)
const restoreReady = ref(false)
const consultationError = ref('')
const fengshuiHandoffLayout = ref<FengshuiLayoutValue | null>(null)
const draftNotice = ref('')
const errorFocusRef = ref<HTMLElement | null>(null)
const documentBoundary = ref<HTMLElement | null>(null)
const shellRef = ref<InstanceType<typeof AppFlowShell> | null>(null)
const { nearBottom, measure: measureNearBottom } = useNearBottom(
  () => shellRef.value?.getMainElement() ?? null,
)
const followVisible = ref(false)
const artifactDockRef = ref<HTMLElement | null>(null)
const castingFormRef = ref<InstanceType<typeof FlowDivinationForm> | null>(null)
const hepanSlotRef = ref<InstanceType<typeof DomainSlotForm> | null>(null)
const fengshuiSlotRef = ref<InstanceType<typeof DomainSlotForm> | null>(null)
const namingSlotRef = ref<InstanceType<typeof DomainSlotForm> | null>(null)
const divinationSlotRef = ref<InstanceType<typeof DomainSlotForm> | null>(null)
let mountToken = 0
let artifactFocusToken = 0

const question = computed({
  get: () => flow.getDraft(),
  set: value => flow.setDraft(value),
})

const primaryError = computed(() => consultationError.value || chat.error)

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
  hydrateDivination,
  retryDivinationChart,
  hepanLoading,
  hepanError,
  hepanInput,
  hepanResult,
  computeHepan,
  reloadHepan,
  saveHepanRecord,
  fengshuiLoading,
  fengshuiError,
  fengshuiInput,
  fengshuiResult,
  fengshuiBirthYear,
  fengshuiGender,
  computeFengshui,
  reloadFengshui,
  saveFengshuiRecord,
  namingLoading,
  namingError,
  namingResult,
  computeNaming,
  reloadNaming,
  profileConsultationBirth,
  resetResults,
} = useConsultationTools({
  selectedProfile,
  personLabel,
  onError: message => {
    consultationError.value = message
  },
  onNotice: announceStatus,
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
  sessionHistory.visibleSessions.value,
)

const latestVisibleSession = computed(() => sessionHistory.latestVisibleSession.value)

const restoreVisible = computed(() =>
  sessionHistory.restoreVisible.value,
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
const activeCastingReceipt = computed(() => {
  const session = chat.activeSession
  if (!session) return null
  if ((session.profile_id || null) !== flow.personId) return null
  if (domainForSession(session) !== flow.domainId) return null
  return session.context?.casting ?? null
})
const showDivinationCasting = computed(() =>
  flow.domainId === 'divination'
  && !castingChart.value
  && !activeCastingReceipt.value,
)
const showHepanForm = computed(() => flow.domainId === 'natal' && routingSeed.value === 'hepan')
const showFengshuiForm = computed(() => flow.domainId === 'fengshui' && routingSeed.value === 'fengshui')
const showNamingForm = computed(() => flow.domainId === 'naming' && routingSeed.value === 'naming')
const showFocusedForm = computed(() =>
  showDivinationCasting.value
  || showHepanForm.value
  || showFengshuiForm.value
  || showNamingForm.value,
)
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
  setDesktopMode,
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
  const value = firstQueryValue(route.query.seed)
    || firstQueryValue(route.query.service)
    || firstQueryValue(route.query.scenario)
  return normalizeLegacyScenario(value)
})

const entrySource = computed<'direct' | 'legacy_redirect' | 'marketing' | 'payment'>(() => {
  if (isLegacyAppRedirect()) return 'legacy_redirect'
  if ([
    'order_id',
    'payment_intent_id',
    'subscription_id',
    'subscription_state',
  ].some(key => firstQueryValue(route.query[key]))) return 'payment'
  if (firstQueryValue(route.query.service) || firstQueryValue(route.query.scenario) || firstQueryValue(route.query.utm_source)) return 'marketing'
  return 'direct'
})

function isLegacyAppRedirect(): boolean {
  return (history.state as { likiAppFlowLegacyRedirect?: unknown } | null)
    ?.likiAppFlowLegacyRedirect === true
}

const hepanArtifact = computed(() =>
  artifacts.value.find(artifact => artifact.kind === 'hepan' && artifact.state === 'ready') ?? null,
)
const fengshuiArtifact = computed(() =>
  artifacts.value.find(artifact => artifact.kind === 'fengshui' && artifact.state === 'ready') ?? null,
)

const focusedFormDirty = computed(() =>
  [hepanSlotRef.value, fengshuiSlotRef.value, namingSlotRef.value, divinationSlotRef.value]
    .some(form => form?.dirty === true),
)

useUnloadGuard(computed(() =>
  isSubmitting.value
  || chat.sending
  || focusedFormDirty.value
  || question.value.trim().length > 0,
))

function selectPerson(personId: string) {
  resetResults()
  flow.setPerson(personId)
  telemetry.trackPersonSelect(flow.domainId)
  personSheetOpen.value = false
}

function selectDomain(domainId: AppDomainId) {
  flow.setDomain(domainId)
  telemetry.trackDomainSelect(domainId)
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

function askDivinationWithoutProfile() {
  if (flow.domainId === 'divination') return
  resetResults()
  consultationError.value = ''
  flow.setDomain('divination')
  telemetry.trackDomainSelect('divination')
}

function openHuangliCompatibility() {
  void router.push({
    path: '/classic/huangli',
    query: route.query.locale ? { locale: firstQueryValue(route.query.locale) } : {},
  })
}

function markFocusedFormsClean() {
  for (const form of [hepanSlotRef.value, fengshuiSlotRef.value, namingSlotRef.value, divinationSlotRef.value]) {
    if (form) form.dirty = false
  }
}

async function syncFocusedFormStep() {
  if (!showFocusedForm.value) return
  if (firstQueryValue(route.query.step) === 'form') return
  await router.replace({
    path: '/',
    query: { ...route.query, step: 'form' },
    hash: route.hash,
  })
}

async function exitFocusedForm() {
  if (focusedFormDirty.value) {
    const discard = await confirmAction({
      title: t('flow.form.unsavedTitle'),
      message: t('flow.form.unsavedMessage'),
      confirmText: t('flow.form.discard'),
    })
    if (!discard) return
  }

  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    const normalized = firstQueryValue(value)
    if (normalized && !['seed', 'service', 'scenario', 'step'].includes(key)) query[key] = normalized
  }
  markFocusedFormsClean()
  resetResults()
  consultationError.value = ''
  await router.replace({ path: '/', query, hash: route.hash })
}

function handleContextAction(id: string) {
  const classicPaths: Record<string, string> = {
    partner: '/classic/partners',
    history: '/classic/chat',
    orders: '/classic/orders',
    account: '/classic/account',
  }
  if (id === 'me') {
    void router.push({ name: 'app-me' })
    return
  }
  if (id === 'classic') {
    void router.push({ name: 'dashboard' })
    return
  }
  if (classicPaths[id]) void router.push(classicPaths[id]!)
}

function startNewConsultation() {
  chat.stopGeneration()
  chat.resetActive()
  flow.clearDraft()
  consultationError.value = ''
  domainSheetOpen.value = false
}

const restoreLatest = sessionHistory.restoreLatest
const latestRestorableSession = sessionHistory.latestRestorableSession

function mostRecentlyUsedProfileId(): string | null {
  return [...profiles.list].sort((left, right) => {
    const leftAt = left.last_used_at ?? left.updated_at
    const rightAt = right.last_used_at ?? right.updated_at
    return rightAt.localeCompare(leftAt)
  })[0]?.id ?? null
}

function nextFlowQuery(domainId: AppDomainId): Record<string, string> {
  const query: Record<string, string> = {}
  const locale = firstQueryValue(route.query.locale)
  if (locale) query.locale = locale

  // Payment-return facts stay available until sales.handlePaymentReturn has
  // consumed them; they are not consultation context and never become seeds.
  for (const key of [
    'order_id',
    'payment_intent_id',
    'subscription_id',
    'subscription_state',
  ]) {
    const value = firstQueryValue(route.query[key])
    if (value) query[key] = value
  }

  query.domain = domainId
  const seed = routingSeed.value
  if (seed && domainForScenario(seed).id === domainId) query.seed = seed
  return query
}

async function syncDomainQuery(domainId: AppDomainId) {
  await router.replace({
    path: '/',
    query: nextFlowQuery(domainId),
    hash: route.hash,
  })
}

function purchaseConsultation(billingMode: 'one_time' | 'subscription') {
  void sales.buy(billingMode, routingSeed.value ?? flow.currentDomain.baseSkill)
}

function projectBridgeFailure(result: Extract<LegacyIntentBridgeResult, { status: 'cross-domain' | 'pending-backend' | 'invalid' }>) {
  telemetry.trackBridge({
    domainId: flow.domainId,
    seed: routingSeed.value,
    result,
  })
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

async function submitQuestion() {
  const content = question.value.trim()
  if (!content || chat.sending || isSubmitting.value) return

  await submit.run(async () => {
    draftNotice.value = ''
    await submitConsultation(content)
  })
}

async function submitConsultation(content: string) {
  consultationError.value = ''

  if (flow.domainId === 'divination' && casting.value && castingChart.value) {
    await submit.submitFocusedForm({
      skill: 'divination',
      context: {
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
      },
      matchesContext: session => session.context?.casting?.casting_id === casting.value?.casting_id,
      content,
      source: 'casting',
    })
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
    await submit.submitFocusedForm({
      skill: 'naming',
      context,
      matchesContext: session => session.context?.scenario === 'naming'
        && sameConsultationBirth(session.context.birth, context.birth),
      content,
    })
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
    await submit.submitFocusedForm({
      skill: 'fengshui',
      context,
      matchesContext: session => session.context?.scenario === 'fengshui'
        && sameConsultationLayout(session.context.layout, context.layout),
      content,
    })
    return
  }

  if (flow.domainId === 'natal' && hepanResult.value && hepanInput.value) {
    const context: ConsultationContext = {
      version: 1,
      scenario: 'hepan',
      second_birth: consultationBirthFromFields(hepanInput.value.secondBirth),
      third_party_consent: true,
    }
    await submit.submitFocusedForm({
      skill: 'hepan',
      context,
      matchesContext: session => session.context?.scenario === 'hepan'
        && sameConsultationBirth(session.context.second_birth, context.second_birth),
      content,
    })
    return
  }

  const existing = latestVisibleSession.value
  if (existing) {
    await submit.openMatchingAndSend(session => session.id === existing.id, content, 'composer')
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
  telemetry.trackBridge({
    domainId: flow.domainId,
    seed: routingSeed.value,
    result: bridgeResult,
  })

  await submit.createAndSendWithSession(bridgeResult.skill, undefined, content, 'composer')
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
  telemetry.trackRecovery(flow.domainId)
  await Promise.allSettled([
    profiles.loadList('', true),
    entitlements.load(true),
    chat.loadSessions(true),
    sales.loadProducts(),
  ])
}

watch([() => flow.personId, () => flow.domainId], ([, domain], [, previousDomain]) => {
  if (!restoreReady.value) return
  resetResults()
  consultationError.value = ''
  markFocusedFormsClean()
  if (domain !== previousDomain) void syncDomainQuery(domain)
  void restoreLatest()
})

watch(nearBottom, (value) => {
  followVisible.value = !value && chat.messages.length > 0
})

watch(workspaceFailure, (failure) => {
  if (failure) telemetry.trackFailure(flow.domainId, failure)
})

// Artifact computation completes in the background (deterministic engines or
// session context projection); announce readiness so screen-reader users know
// a reference capsule appeared without scanning the dock.
watch(() => artifacts.value.filter(artifact => artifact.state === 'ready').length, (count, previous) => {
  if (count > (previous ?? 0)) announceStatus(t('flow.artifactReady'))
})

watch([hepanResult, fengshuiResult, namingResult, castingChart], async () => {
  await nextTick()
  markFocusedFormsClean()
  const hasResult = Boolean(hepanResult.value || fengshuiResult.value || namingResult.value || castingChart.value)
  if (hasResult && showFocusedForm.value) {
    const query: Record<string, string> = {}
    for (const [key, value] of Object.entries(route.query)) {
      const normalized = firstQueryValue(value)
      if (normalized && !['seed', 'service', 'scenario', 'step'].includes(key)) query[key] = normalized
    }
    await router.replace({ path: '/', query, hash: route.hash })
  }
})

watch(primaryError, async (value) => {
  if (!value) return
  await nextTick()
  errorFocusRef.value?.focus({ preventScroll: true })
})

watch(() => `${chat.messages.length}:${chat.messages.at(-1)?.content.length ?? 0}`, () => {
  if (nearBottom.value) shellRef.value?.scrollToBottom()
  else followVisible.value = true
})

onMounted(async () => {
  const token = ++mountToken
  const requestedDomain = normalizeAppDomain(firstQueryValue(route.query.domain))
    ?? marketingScenarioToDomain(
      firstQueryValue(route.query.service)
      || firstQueryValue(route.query.scenario)
      || firstQueryValue(route.query.seed),
    )
  if (requestedDomain) flow.setDomain(requestedDomain)
  // Marketing divination handoff: keep the original casting receipt, then load
  // its chart in the App instead of drawing a new hexagram. Wait one tick so the
  // domain-change watcher's reset does not wipe the hydrated receipt.
  if (flow.domainId === 'divination') {
    await nextTick()
    const handoff = consumeConsultationHandoff<ConsultationContext>('divination')
    if (handoff) void hydrateDivination(handoff)
  }
  // Marketing fengshui handoff: prefill the layout form, then let the user
  // select a profile and compute as usual.
  if (flow.domainId === 'fengshui') {
    await nextTick()
    const handoff = consumeConsultationHandoff<ConsultationContext>('fengshui')
    if (handoff?.layout) {
      fengshuiHandoffLayout.value = {
        zuoShan: handoff.layout.zuo_shan,
        periodDate: handoff.layout.period_date,
        doorGua: handoff.layout.door_gua ?? '',
        masterGua: handoff.layout.master_gua ?? '',
        stoveGua: handoff.layout.stove_gua ?? '',
      }
    }
  }
  await Promise.allSettled([
    profiles.loadList(),
    entitlements.load(),
    chat.loadSessions(true),
    sales.loadProducts(),
  ])
  if (token !== mountToken) return

  // URL context is authoritative. Without it, the newest active session is
  // authoritative and projects its person/domain back into flow state.
  const bootCandidate = latestRestorableSession(requestedDomain ?? null)
  if (!requestedDomain && bootCandidate) flow.setDomain(domainForSession(bootCandidate))
  if (!requestedDomain && !bootCandidate && !profiles.list.length) flow.setDomain('divination')

  if (bootCandidate) flow.setPerson(bootCandidate.profile_id || null)
  else if (!flow.personId) flow.setPerson(mostRecentlyUsedProfileId())
  telemetry.trackEntry({
    domainId: flow.domainId,
    seed: routingSeed.value,
    source: entrySource.value,
    restorable: Boolean(bootCandidate),
  })
  await restoreLatest()
  restoreReady.value = true
  await syncFocusedFormStep()

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
      telemetry.trackRestore({
        domainId: flow.domainId,
        source: 'payment',
        restored: chat.activeSession?.id === sessionId,
      })
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

/* Programmatic focus identifies the error group without adding a second
   visible focus ring around an already visible alert. */
.flow-error-focus:focus {
  outline: none;
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
</style>
