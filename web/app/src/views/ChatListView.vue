<template>
  <section class="page-stack">
    <AppPageHeader
      :title="$t('router.chat')"
      :subtitle="$t('chat.subtitle')"
    />

    <el-alert
      v-if="chat.error"
      :title="chat.error"
      type="error"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="sales.paymentMessage"
      :title="sales.paymentMessage"
      :type="sales.paymentState === 'confirmed' ? 'success' : sales.paymentState === 'unpaid' ? 'warning' : 'info'"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="sales.actionError"
      :title="sales.actionError"
      type="error"
      :closable="false"
      show-icon
    />

    <el-skeleton
      v-if="sales.initialLoading"
      :rows="7"
      animated
      aria-busy="true"
    />

    <el-result
      v-else-if="sales.showProductError"
      icon="error"
      :title="$t('chat.productsFailed')"
      :sub-title="sales.productsError"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="sales.loadProducts"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>

    <template v-else>
      <ProductPurchaseCard
        v-if="sales.chatProduct && !sales.chatProduct.purchased"
        id="chat-purchase"
        :product="sales.chatProduct"
        :buying="sales.buying"
        @purchase="billingMode => sales.buy(billingMode, selectedIntent || pendingIntent?.id || 'natal')"
      />

      <el-card
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('chat.selectBirthProfile') }}
          </h2>
        </template>
        <div ref="profilePickerEl">
          <BirthProfilePicker
            :profiles="profiles.list"
            :model-value="activeBirthProfileId"
            @update:model-value="activeBirthProfileId = $event"
          />
        </div>
        <el-alert
          v-if="!activeBirthProfileId"
          :title="$t('chat.selectBirthProfileFirst')"
          type="info"
          class="inline-alert"
          :closable="false"
          show-icon
        />
      </el-card>

      <el-card
        id="chat-intent"
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('chat.selectIntent') }}
          </h2>
        </template>
        <IntentEntryList
          :items="intentEntries"
          @select="selectIntent"
        />
      </el-card>

      <el-alert
        v-if="pendingIntent"
        :title="$t('chat.pendingIntent', { service: $t(pendingIntent.labelKey) })"
        :description="$t(pendingIntent.descKey)"
        type="info"
        :closable="false"
        show-icon
      />

      <el-card
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('dashboard.recentSessions') }}
          </h2>
        </template>
        <el-skeleton
          v-if="chat.loadingSessions && !chat.sessions.length"
          :rows="3"
          animated
          aria-busy="true"
        />
        <AppEmptyState
          v-else-if="!chat.sessions.length"
          :description="$t('chat.empty')"
        >
          <el-button
            v-if="!profiles.list.length"
            type="primary"
            tag="router-link"
            to="/birth-profiles/new"
          >
            {{ $t('profiles.create') }}
          </el-button>
          <el-button
            v-else
            type="primary"
            @click="scrollToIntents"
          >
            {{ $t('chat.emptyAction') }}
          </el-button>
        </AppEmptyState>
        <SessionList
          v-else
          :sessions="chat.sessions"
          @open="openSession"
          @rename="renameSession"
          @remove="removeSession"
        />
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { confirmDeleteEntity, promptRenameEntity } from '@/utils/dialogs'
import { firstQueryValue } from '@/utils/query'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useChatStore } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useSalesStore } from '@/stores/sales-store'
import {
  type ConsultationBirth,
  consultationBirthFromFields,
  type ConsultationContext,
  type ConsultationIntent,
} from '@shared/domain/consultation'
import { clearConsultationHandoff, consumeConsultationHandoff, peekConsultationHandoff } from '@shared/utils/handoff'
import type { BirthFieldsValue } from '@shared/components/form/birth-fields'
import { isCurrentLocation } from '@shared/domain/location'
import { intentByIntent, intentById, intentEntryItems } from '@/config/chat'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppEmptyState from '@/components/app/AppEmptyState.vue'
import IntentEntryList from '@/components/app/IntentEntryList.vue'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import SessionList from '@/components/chat/SessionList.vue'
import BirthProfilePicker from '@/components/chat/BirthProfilePicker.vue'
import { useFormDraftsStore } from '@shared/stores/form-drafts'
import type { ChatSession } from '@/api/chat'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const chat = useChatStore()
const entitlements = useEntitlementsStore()
const profiles = useBirthProfilesStore()
const sales = useSalesStore()
const drafts = useFormDraftsStore()

const activeBirthProfileId = ref('')
const creating = ref('')
const selectedIntent = ref('')

const chatPurchased = computed(() => entitlements.purchased)
const profilePickerEl = ref<HTMLElement | null>(null)

const pendingIntent = computed(() => (chat.pendingIntent ? intentByIntent(chat.pendingIntent) : undefined))

// 入口行右侧动作文案：随可用状态给出下一步（开通/查看订单/建档案/开始）。
const intentAction = computed(() => {
  if (!chatPurchased.value) return t('chat.action.activate')
  if (!entitlements.hasSkill('liki')) return t('chat.action.orders')
  if (!activeBirthProfileId.value) return t('profiles.create')
  return t('chat.action.start')
})

const intentEntries = computed(() =>
  intentEntryItems(t).map(item => ({ ...item, action: intentAction.value })),
)

// 选择场景：未开通 → 滚到开通卡；方案未含 → 订单页；缺档案 → 建档案；就绪 → 创建会话。
async function selectIntent(id: string) {
  selectedIntent.value = id
  if (!chatPurchased.value) {
    document.getElementById('chat-purchase')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    return
  }
  if (!entitlements.hasSkill('liki')) {
    void router.push({ name: 'orders' })
    return
  }
  if (!activeBirthProfileId.value) {
    void router.push({ name: 'birth-profile-new' })
    return
  }
  if (isBirthConsultationIntent(id)) {
    const handoff = peekConsultationHandoff<ConsultationContext>(id)
    const handoffBirth = handoff?.birth
    const hasMatchingProfile = handoffBirth
      ? profiles.list.some(profile => profileMatchesConsultationBirth(profile, handoffBirth))
      : false
    if (handoffBirth && !hasMatchingProfile) {
      drafts.set('service', id)
      await router.replace({
        name: 'birth-profile-new',
        query: route.query.locale ? { locale: firstQueryValue(route.query.locale) } : {},
      })
      return
    }
  }
  void create(id)
}

function openSession(session: ChatSession) {
  void router.push({ name: 'chat-session', params: { sessionId: session.id } })
}

// 会话空态的下一步（已有档案时）：滚动到咨询类型列表。
function scrollToIntents() {
  document.getElementById('chat-intent')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function renameSession(session: ChatSession) {
  let value: string | null
  try {
    value = await promptRenameEntity(t('chat.sessions.noun'), session.title)
  } catch (error) {
    chat.error = toUserMessage(error, t('common.actionFailed'))
    return
  }
  if (value === null) return
  try {
    await chat.renameSession(session.id, value)
  } catch (error) {
    chat.error = toUserMessage(error, t('common.actionFailed'))
  }
}

async function removeSession(session: ChatSession) {
  if (!await confirmDeleteEntity(t('chat.sessions.noun'), session.title)) return
  try {
    await chat.removeSession(session.id)
  } catch (error) {
    chat.error = toUserMessage(error, t('common.actionFailed'))
  }
}

async function create(intentId: string) {
  if (!activeBirthProfileId.value || creating.value) return
  creating.value = intentId
  sales.actionError = ''
  try {
    let profile = profiles.list.find(item => item.id === activeBirthProfileId.value)
    if (isBirthConsultationIntent(intentId)) {
      const handoff = peekConsultationHandoff<ConsultationContext>(intentId)
      const handoffBirth = handoff?.birth
      const matchingProfile = handoffBirth
        ? profiles.list.find(item => profileMatchesConsultationBirth(item, handoffBirth))
        : undefined
      if (matchingProfile) {
        activeBirthProfileId.value = matchingProfile.id
        profile = matchingProfile
      }
    }
    const session = await chat.createSession(
      activeBirthProfileId.value,
      intentId,
      profile ? consultationContextFor(intentId as ConsultationIntent, profile) ?? undefined : undefined,
    )
    await router.push({ name: 'chat-session', params: { sessionId: session.id } })
  } catch (error) {
    sales.actionError = toUserMessage(error, t('chat.createFailed'))
  } finally {
    creating.value = ''
  }
}

function birthFieldsOf(profile: (typeof profiles.list)[number]): BirthFieldsValue | null {
  const { gender, date, time, location } = profile.birth
  if (!date || !time || location.longitude == null || !location.timezone) return null
  const current = isCurrentLocation(location)
  return {
    gender,
    date,
    time,
    location: {
      place: current ? location.place : '',
      ...(current && location.place_id ? { place_id: location.place_id } : {}),
      longitude: location.longitude,
      timezone: location.timezone,
      source: 'saved',
    },
  }
}

function consultationContextFor(intentId: ConsultationIntent, profile: (typeof profiles.list)[number]): ConsultationContext | null {
  const handoff = peekConsultationHandoff<ConsultationContext>(intentId)
  if (handoff) {
    const profileMatches = !handoff.birth || profileMatchesConsultationBirth(profile, handoff.birth)
    if (profileMatches) {
      consumeConsultationHandoff<ConsultationContext>(intentId)
      return handoff
    }
    clearConsultationHandoff()
  }
  // Only birth-based intents can be built from a saved profile. Tool
  // intents (divination/huangli/fengshui/hepan) must use their handoff; a
  // fabricated natal context would be rejected by the server and is exactly
  // the wrong-agent class of bug this guard prevents.
  if (!isBirthConsultationIntent(intentId)) return null
  const fields = birthFieldsOf(profile)
  if (!fields) return null
  const birth = consultationBirthFromFields(fields)
  if (intentId === 'liunian') {
    return {
      version: 1,
      intent: 'liunian',
      birth,
      year: new Date().getFullYear(),
    }
  }
  return {
    version: 1,
    intent: intentId,
    birth,
  }
}

function profileMatchesConsultationBirth(profile: (typeof profiles.list)[number], birth: ConsultationBirth): boolean {
  const actual = profile.birth
  const actualLocation = actual.location
  const expectedLocation = birth.location
  if (actualLocation.longitude == null) return false

  return actual.gender === birth.gender
    && actual.date === birth.date
    && actual.time === birth.time
    && actualLocation.timezone === expectedLocation.timezone
    && Math.abs(actualLocation.longitude - expectedLocation.longitude) < 0.000001
}

function isBirthConsultationIntent(value: string): value is ConsultationIntent {
  return value === 'natal' || value === 'liunian' || value === 'naming'
}

onMounted(async () => {
  try {
    void chat.loadSessions()
    await sales.refresh()
    await profiles.loadList()
    const preset = firstQueryValue(route.query.profile)
    if (preset && profiles.list.some(p => p.id === preset)) {
      activeBirthProfileId.value = preset
    } else if (profiles.list.length) {
      activeBirthProfileId.value = profiles.list[0].id
    }
    // 服务入口 → 会话打开后自动发出首条意图（由 ChatSessionView 消费）。
    const presetIntent = firstQueryValue(route.query.service) ?? firstQueryValue(route.query.intent)
    const svc = intentById(presetIntent)
    if (svc) chat.pendingIntent = t(svc.intentKey)
    if (svc && !preset && isBirthConsultationIntent(svc.id)) {
      const handoff = peekConsultationHandoff<ConsultationContext>(svc.id)
      const handoffBirth = handoff?.birth
      const matchingProfile = handoffBirth
        ? profiles.list.find(profile => profileMatchesConsultationBirth(profile, handoffBirth))
        : undefined
      if (matchingProfile) {
        activeBirthProfileId.value = matchingProfile.id
      }
      else if (handoffBirth) {
        drafts.set('service', svc.id)
        await router.replace({
          name: 'birth-profile-new',
          query: route.query.locale ? { locale: firstQueryValue(route.query.locale) } : {},
        })
        return
      }
    }

    // 支付回调处理：store 返回结果，由 view 决定路由。
    const paymentResult = await sales.handlePaymentReturn(route.query)
    if (paymentResult.action === 'enter_chat') {
      const sessionId = await sales.enterInterpretation(activeBirthProfileId.value, paymentResult.intentId)
      if (sessionId) {
        await router.replace({ name: 'chat', query: {} })
        await router.push({ name: 'chat-session', params: { sessionId } })
      }
    } else if (paymentResult.action === 'show_message') {
      await router.replace({ name: 'chat', query: {} })
    }
  } catch (error) {
    // 网络/初始化失败不崩溃——store 内部已有 error state 展示。
    console.error('[ChatList] 初始化失败:', error)
  }
})
</script>

<style scoped>
.inline-alert { margin-bottom: 0; }
</style>
