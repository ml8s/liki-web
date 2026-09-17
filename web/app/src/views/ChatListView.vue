<template>
  <section>
    <AppPageHeader
      title="Liki Chat"
      subtitle="八字、合盘、起名、问卦、风水——一个入口，专业解读。"
    >
      <template #actions>
        <el-button :loading="commerce.refreshing" @click="commerce.refresh">刷新</el-button>
      </template>
    </AppPageHeader>

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

      <AppAlert
        v-if="pendingService"
        :title="`准备开始：${pendingService.label}`"
        :description="pendingService.desc"
        type="info"
      />

      <h2>最近会话</h2>
      <AppEmpty v-if="!chat.sessions.length" description="还没有会话，先在上方选择档案与领域开启咨询" />
      <SessionList
        v-else
        :sessions="displaySessions"
        @open="openSession"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
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
  await commerce.refresh()
  await profiles.loadList()
  const preset = commerce.firstQuery(route.query.profile)
  if (preset && profiles.list.some(p => p.id === preset)) {
    activeProfileId.value = preset
  } else if (profiles.list.length) {
    activeProfileId.value = profiles.list[0].id
  }
  // 服务入口 → 会话打开后自动发出首条意图（由 ChatSessionView 消费）。
  const presetService = commerce.firstQuery(route.query.service)
  const svc = serviceByKey(presetService)
  if (svc) chat.pendingIntent = svc.intent
  await commerce.handlePaymentReturn()
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