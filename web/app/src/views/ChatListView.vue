<template>
  <section>
    <AppPageHeader
      title="Liki Chat"
      subtitle="起名、八字、问卦、风水，一个入口统一咨询。"
    >
      <template #actions>
        <el-button :loading="refreshing" @click="refresh">刷新</el-button>
      </template>
    </AppPageHeader>

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert
      v-if="paymentMessage"
      :title="paymentMessage"
      :type="paymentState === 'confirmed' ? 'success' : paymentState === 'unpaid' ? 'warning' : 'info'"
    />
    <AppAlert v-if="actionError" :title="actionError" type="error" />

    <AppSkeleton v-if="initialLoading" :rows="7" />

    <AppErrorState
      v-else-if="showProductError"
      title="产品加载失败"
      :description="productsError"
      @retry="loadProducts"
    />

    <template v-else>
      <ProductPurchaseCard
        v-if="chatProduct && !chatProduct.purchased"
        :product="chatProduct"
        :buying="buying"
        @purchase="buy"
      />

      <div class="picker-row">
        <ProfilePicker
          :profiles="profiles.list"
          :model-value="activeProfileId"
          @update:model-value="activeProfileId = $event"
        />
        <AppAlert v-if="!activeProfileId" title="请先选择一个档案" type="info" class="inline-alert" />
      </div>

      <div class="skill-grid">
        <SkillCard
          v-for="item in chatSkills"
          :key="item.id"
          :label="item.label"
          :description="item.description"
          :enabled="canStart(item.id)"
          :starting="creating === item.id"
          @start="create(item.id)"
        />
      </div>

      <h2>最近会话</h2>
      <AppEmpty v-if="!chat.sessions.length" description="还没有会话" />
      <SessionList
        v-else
        :sessions="displaySessions"
        @open="openSession"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useProfilesStore } from '@/stores/profiles'
import { checkoutChatProduct, fetchChatProducts, type ChatProduct } from '@/api/commerce'
import { chatSkills } from '@/config/chat'
import AppAlert from '@/components/app/AppAlert.vue'
import AppEmpty from '@/components/app/AppEmpty.vue'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'
import ProductPurchaseCard from '@/components/commerce/ProductPurchaseCard.vue'
import SessionList from '@/components/chat/SessionList.vue'
import SkillCard from '@/components/chat/SkillCard.vue'
import ProfilePicker from '@/components/chat/ProfilePicker.vue'
import type { ChatSession } from '@/api/chat'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const entitlements = useEntitlementsStore()
const profiles = useProfilesStore()

const activeProfileId = ref('')
const creating = ref('')
const buying = ref(false)
const refreshing = ref(false)
const loadingProducts = ref(false)
const products = ref<ChatProduct[]>([])
const productsError = ref('')
const actionError = ref('')
const paymentState = ref<'idle' | 'verifying' | 'confirmed' | 'unpaid'>('idle')
const paymentMessage = ref('')

function firstQuery(value: unknown): string {
  if (Array.isArray(value)) return String(value[0] ?? '')
  return String(value ?? '')
}

async function wait(ms: number) {
  await new Promise(resolve => setTimeout(resolve, ms))
}

async function clearPaymentQuery() {
  await router.replace({ path: '/chat', query: {} })
}

async function handlePaymentReturn() {
  const orderId = firstQuery(route.query.order_id)
  const paymentIntentId = firstQuery(route.query.payment_intent_id)
  const status = firstQuery(route.query.status)
  if (!orderId && !paymentIntentId) return

  if (status === 'paid') {
    paymentState.value = 'verifying'
    paymentMessage.value = '支付已确认，正在开通 Liki Chat…'

    // Webhook and durable fulfillment may finish shortly after the provider
    // redirects the browser back to the SPA. Poll briefly instead of exposing
    // a false "not purchased" state.
    for (let attempt = 0; attempt < 12; attempt += 1) {
      await refresh()
      if (chatPurchased.value) {
        paymentState.value = 'confirmed'
        paymentMessage.value = '支付成功，Liki Chat 已开通。'
        await clearPaymentQuery()
        // V4：支付成功后直达解读——有已选档案则自动用默认 skill 建会话并跳转。
        await enterInterpretation()
        return
      }
      await wait(1000 * (attempt + 1))
    }

    paymentState.value = 'verifying'
    paymentMessage.value = '支付已收到，权益开通仍在处理。请稍后刷新本页。'
    await clearPaymentQuery()
    return
  }

  if (status === 'unpaid') {
    paymentState.value = 'unpaid'
    paymentMessage.value = '支付尚未完成。如已完成付款，请稍后刷新订单状态。'
  }

  await clearPaymentQuery()
}

const chatProduct = computed(() => products.value.find(item => item.id === 'liki-chat'))
const chatPurchased = computed(() => entitlements.allowedSkills.length === 4)
const initialLoading = computed(() =>
  !products.value.length && (loadingProducts.value || entitlements.loading || chat.loadingSessions),
)
const showProductError = computed(() => !!productsError.value && !products.value.length)

const displaySessions = computed(() => chat.sessions.map(session => ({
  ...session,
  skillLabel: skillLabel(session.skill),
} satisfies ChatSession & { skillLabel: string })))

function canStart(skill: string) {
  return chatPurchased.value && entitlements.hasSkill(skill) && !!activeProfileId.value
}

function openSession(session: ChatSession) {
  void router.push(`/chat/${session.id}`)
}

async function create(skill: string) {
  if (!activeProfileId.value) return
  creating.value = skill
  actionError.value = ''
  try {
    const session = await chat.createSession(activeProfileId.value, skill)
    await router.push(`/chat/${session.id}`)
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : '创建会话失败'
  } finally {
    creating.value = ''
  }
}

// V4：支付返回直达解读——有已选档案则用默认 skill（八字）建会话并跳转。
async function enterInterpretation() {
  if (!activeProfileId.value) {
    await profiles.loadList()
    activeProfileId.value = profiles.list[0]?.id ?? ''
    if (!activeProfileId.value) return
  }
  try {
    const session = await chat.createSession(activeProfileId.value, 'liki-bazi')
    await router.replace(`/chat/${session.id}`)
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : '创建会话失败'
  }
}

async function loadProducts() {
  loadingProducts.value = true
  productsError.value = ''
  try {
    products.value = await fetchChatProducts()
  } catch (error) {
    productsError.value = error instanceof Error ? error.message : '加载产品失败'
  } finally {
    loadingProducts.value = false
  }
}

async function buy() {
  if (!chatProduct.value) return
  buying.value = true
  actionError.value = ''
  try {
    const checkout = await checkoutChatProduct(chatProduct.value.id)
    window.location.href = checkout.checkout_url
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : '发起支付失败'
    await refresh()
  } finally {
    buying.value = false
  }
}

async function refresh() {
  refreshing.value = true
  actionError.value = ''
  try {
    await Promise.all([entitlements.load(true), chat.loadSessions(), loadProducts()])
  } finally {
    refreshing.value = false
  }
}

onMounted(async () => {
  await refresh()
  await profiles.loadList()
  const preset = firstQuery(route.query.profile)
  if (preset && profiles.list.some(p => p.id === preset)) {
    activeProfileId.value = preset
  } else if (profiles.list.length) {
    activeProfileId.value = profiles.list[0].id
  }
  await handlePaymentReturn()
})
</script>

<style scoped>
.picker-row { display: flex; align-items: center; gap: 14px; margin-bottom: 20px; flex-wrap: wrap; }
.inline-alert { margin-bottom: 0; }
.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}
h2 { margin: 0 0 14px; }
</style>
