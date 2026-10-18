<template>
  <section>
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
      icon="warning"
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
            :starting="creating === item.id"
            @start="create(item.id)"
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
        <el-empty
          v-if="!chat.sessions.length"
          :description="$t('chat.empty')"
        >
          <el-button
            type="primary"
            tag="router-link"
            to="/birth-profiles/new"
          >
            {{ $t('profiles.create') }}
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
	    }
	  }
  if (!entitlements.hasSkill('liki')) {
	    return {
	      enabled: false,
	      reason: t('chat.state.notIncluded'),
	      status: t('chat.status.notIncluded'),
	      actionLabel: t('chat.action.orders'),
	      actionTo: '/orders',
	    }
  }
  if (!activeBirthProfileId.value) {
	    return {
	      enabled: false,
	      reason: t('chat.state.profileRequired'),
	      status: t('chat.status.profileRequired'),
	      actionLabel: t('profiles.create'),
	      actionTo: '/birth-profiles/new',
	    }
  }
	  return {
	    enabled: true,
	    reason: '',
	    status: t('chat.status.available'),
	    actionLabel: t('chat.action.start'),
	    actionTo: '',
	  }
}

function openSession(session: ChatSession) {
  void router.push(`/chat/${session.id}`)
}

async function renameSession(session: ChatSession) {
  let value: string
  try {
    const result = await ElMessageBox.prompt(
      t('chat.sessions.renamePrompt'),
      t('chat.sessions.renameTitle'),
      {
        inputValue: session.title,
        inputPlaceholder: t('chat.sessions.renamePlaceholder'),
        inputValidator: (raw: string) => {
          const title = raw.trim()
          if (!title) return t('chat.sessions.renameRequired')
          if (title.length > 60) return t('chat.sessions.renameTooLong')
          return true
        },
        confirmButtonText: t('common.save'),
        cancelButtonText: t('common.cancel'),
      },
    )
    value = result.value.trim()
  } catch (error) {
    // prompt 取消以 'cancel'/'close' 拒绝，不算错误。
    if (error === 'cancel' || error === 'close') return
    chat.error = toUserMessage(error, t('chat.sessions.actionFailed'))
    return
  }
  try {
    await chat.renameSession(session.id, value)
  } catch (error) {
    chat.error = toUserMessage(error, t('chat.sessions.actionFailed'))
  }
}

async function removeSession(session: ChatSession) {
  try {
    await ElMessageBox.confirm(
      t('chat.sessions.deleteConfirm', { title: session.title }),
      t('chat.sessions.deleteTitle'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  try {
    await chat.removeSession(session.id)
  } catch (error) {
    chat.error = toUserMessage(error, t('chat.sessions.actionFailed'))
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
.section-card { margin-bottom: 16px; }
.inline-alert { margin-bottom: 0; }
.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}
</style>
