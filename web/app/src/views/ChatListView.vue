<template>
  <section>
    <AppPageHeader title="Liki Chat" subtitle="八字、合盘、起名、问卦、风水——一个入口，专业解读。" />

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert
      v-if="commerce.paymentMessage"
      :title="commerce.paymentMessage"
      :type="commerce.paymentState === 'confirmed' ? 'success' : commerce.paymentState === 'unpaid' ? 'warning' : 'info'"
    />
    <AppAlert v-if="commerce.actionError" :title="commerce.actionError" type="error" />

    <AppSkeleton v-if="commerce.initialLoading" :rows="7" />

    <AppErrorState
      v-else-if="commerce.showProductError"
      title="产品加载失败"
      :description="commerce.productsError"
      @retry="commerce.loadProducts"
    />

    <template v-else>
      <ProductPurchaseCard
        v-if="commerce.chatProduct && !commerce.chatProduct.purchased"
        :product="commerce.chatProduct"
        :buying="commerce.buying"
        @purchase="commerce.buy"
      />

      <el-card class="section-card" shadow="never">
        <template #header><span class="card-title">选择档案</span></template>
        <ProfilePicker
          :profiles="profiles.list"
          :model-value="activeProfileId"
          @update:model-value="activeProfileId = $event"
        />
        <AppAlert v-if="!activeProfileId" title="请先选择一个档案" type="info" class="inline-alert" />
      </el-card>

      <el-card class="section-card" shadow="never">
        <template #header><span class="card-title">选择解读领域</span></template>
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
      </el-card>

      <AppAlert
        v-if="pendingService"
        :title="`准备开始：${pendingService.label}`"
        :description="pendingService.desc"
        type="info"
      />

      <el-card class="section-card" shadow="never">
        <template #header><span class="card-title">最近会话</span></template>
        <AppEmpty v-if="!chat.sessions.length" description="还没有会话，选择档案与领域后开启解读" />
        <SessionList
          v-else
          :sessions="displaySessions"
          @open="openSession"
        />
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { toFirstQueryString } from '@/utils/format'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useProfilesStore } from '@/stores/profiles'
import { useCommerceStore } from '@/stores/commerce'
import { chatSkills, serviceByIntent, serviceByKey } from '@/config/chat'
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
const commerce = useCommerceStore()

const activeProfileId = ref('')
const creating = ref('')

const chatPurchased = computed(() => entitlements.allowedSkills.length > 0)

const pendingService = computed(() => (chat.pendingIntent ? serviceByIntent(chat.pendingIntent) : undefined))

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
  commerce.actionError = ''
  try {
    const session = await chat.createSession(activeProfileId.value, skill)
    await router.push(`/chat/${session.id}`)
  } catch (error) {
    commerce.actionError = toUserMessage(error, '创建会话失败')
  } finally {
    creating.value = ''
  }
}

onMounted(async () => {
  try {
    await commerce.refresh()
    await profiles.loadList()
    const preset = toFirstQueryString(route.query.profile)
    if (preset && profiles.list.some(p => p.id === preset)) {
      activeProfileId.value = preset
    } else if (profiles.list.length) {
      activeProfileId.value = profiles.list[0].id
    }
    // 服务入口 → 会话打开后自动发出首条意图（由 ChatSessionView 消费）。
    const presetService = toFirstQueryString(route.query.service)
    const svc = serviceByKey(presetService)
    if (svc) chat.pendingIntent = svc.intent

    // 支付回调处理：store 返回结果，由 view 决定路由。
    const paymentResult = await commerce.handlePaymentReturn(route.query)
    if (paymentResult.action === 'enter_chat') {
      const sessionId = await commerce.enterInterpretation()
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
.section-card { border-radius: 12px; margin-bottom: 16px; }
.card-title { font-weight: 600; }
.inline-alert { margin-bottom: 0; }
.skill-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}
</style>
