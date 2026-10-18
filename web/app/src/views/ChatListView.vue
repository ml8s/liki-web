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
        @purchase="billingMode => sales.buy(billingMode)"
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
        id="chat-domain"
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('chat.selectDomain') }}
          </h2>
        </template>
        <div class="skill-grid">
          <SkillCard
            v-for="item in chatSkills"
            :key="item.id"
            :label="$t(item.labelKey)"
            :description="$t(item.descriptionKey)"
            :enabled="canStart()"
            :blocked-reason="skillState().reason"
            :status-label="skillState().status"
            :action-label="skillState().actionLabel"
            :action-to="skillState().actionTo"
            :action-scroll-to="skillState().actionScrollTo"
            :starting="creating === item.id"
            @start="create(item.id)"
            @anchor="handleSkillAnchor"
          />
        </div>
      </el-card>

      <el-alert
        v-if="pendingService"
        :title="$t('chat.pendingService', { service: $t(pendingService.labelKey) })"
        :description="$t(pendingService.descKey)"
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
        <el-empty
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
            @click="scrollToSkills"
          >
            {{ $t('chat.emptyAction') }}
          </el-button>
        </el-empty>
        <SessionList
          v-else
          :sessions="displaySessions"
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
import { useChatStore, skillLabel } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useSalesStore } from '@/stores/sales-store'
import { chatSkills, serviceByIntent, serviceByKey } from '@/config/chat'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import ProductPurchaseCard from '@/components/sales/ProductPurchaseCard.vue'
import SessionList from '@/components/chat/SessionList.vue'
import SkillCard from '@/components/chat/SkillCard.vue'
import BirthProfilePicker from '@/components/chat/BirthProfilePicker.vue'
import type { ChatSession } from '@/api/chat'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const chat = useChatStore()
const entitlements = useEntitlementsStore()
const profiles = useBirthProfilesStore()
const sales = useSalesStore()

const activeBirthProfileId = ref('')
const creating = ref('')

const chatPurchased = computed(() => entitlements.purchased)
const profilePickerEl = ref<HTMLElement | null>(null)

const pendingService = computed(() => (chat.pendingIntent ? serviceByIntent(chat.pendingIntent) : undefined))

const displaySessions = computed(() => chat.sessions.map(session => ({
  ...session,
  skillLabel: skillLabel(session.expert),
} satisfies ChatSession & { skillLabel: string })))

function canStart() {
  // 权益校验仍以统一 skill「liki」为准；专家只是部署路由维度。
  return chatPurchased.value && entitlements.hasSkill('liki') && !!activeBirthProfileId.value
}

function skillState() {
	  if (!chatPurchased.value) {
	    return {
	      enabled: false,
	      reason: t('chat.state.inactive'),
	      status: t('chat.status.inactive'),
	      actionLabel: t('chat.action.activate'),
	      actionTo: '',
	      actionScrollTo: '#chat-purchase',
	    }
	  }
  if (!entitlements.hasSkill('liki')) {
	    return {
	      enabled: false,
	      reason: t('chat.state.notIncluded'),
	      status: t('chat.status.notIncluded'),
	      actionLabel: t('chat.action.orders'),
	      actionTo: '/orders',
	      actionScrollTo: '',
	    }
  }
  if (!activeBirthProfileId.value) {
	    return {
	      enabled: false,
	      reason: t('chat.state.profileRequired'),
	      status: t('chat.status.profileRequired'),
	      actionLabel: t('profiles.create'),
	      actionTo: '/birth-profiles/new',
	      actionScrollTo: '',
	    }
  }
	  return {
	    enabled: true,
	    reason: '',
	    status: t('chat.status.available'),
	    actionLabel: t('chat.action.start'),
	    actionTo: '',
	    actionScrollTo: '',
	  }
}

function openSession(session: ChatSession) {
  void router.push(`/chat/${session.id}`)
}

// 会话空态的下一步（已有档案时）：滚动到领域选择卡。
function scrollToSkills() {
  document.getElementById('chat-domain')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function handleSkillAnchor(selector: string) {
  document.querySelector(selector)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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

async function create(expert: string) {
  if (!activeBirthProfileId.value) return
  creating.value = expert
  sales.actionError = ''
  try {
    const session = await chat.createSession(activeBirthProfileId.value, expert)
    await router.push(`/chat/${session.id}`)
  } catch (error) {
    sales.actionError = toUserMessage(error, t('chat.createFailed'))
  } finally {
    creating.value = ''
  }
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
    const presetService = firstQueryValue(route.query.service)
    const svc = serviceByKey(presetService)
    if (svc) chat.pendingIntent = t(svc.intentKey)

    // 支付回调处理：store 返回结果，由 view 决定路由。
    const paymentResult = await sales.handlePaymentReturn(route.query)
    if (paymentResult.action === 'enter_chat') {
      const sessionId = await sales.enterInterpretation(activeBirthProfileId.value)
      if (sessionId) {
        await router.replace({ path: '/chat', query: {} })
        await router.push(`/chat/${sessionId}`)
      }
    } else if (paymentResult.action === 'show_message') {
      await router.replace({ path: '/chat', query: {} })
    }
  } catch (error) {
    // 网络/初始化失败不崩溃——store 内部已有 error state 展示。
    console.error('[ChatList] 初始化失败:', error)
  }
})
</script>

<style scoped>
.inline-alert { margin-bottom: 0; }
.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}
</style>
