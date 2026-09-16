<template>
  <section>
    <AppPageHeader title="工作台" subtitle="查看排盘，或继续上次咨询。" />

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert v-else-if="profiles.error" :title="profiles.error" type="error" />

    <AppSkeleton v-if="loading" :rows="4" />

    <template v-else-if="profiles.list.length">
      <!-- 主行动：优先继续咨询，否则查看最近档案的排盘 -->
      <el-card v-if="recentSession" class="primary" shadow="never">
        <template #header><span class="card-title">继续上次咨询</span></template>
        <div class="primary-item">
          <div class="primary-title">{{ recentSession.title }}</div>
          <div class="primary-meta">{{ skillLabel(recentSession.skill) }} · {{ formatTime(recentSession.last_message_at ?? recentSession.updated_at) }}</div>
        </div>
        <el-button type="primary" size="large" @click="router.push(`/chat/${recentSession.id}`)">继续</el-button>
      </el-card>

      <el-card v-else-if="latestProfile" class="primary" shadow="never">
        <template #header><span class="card-title">查看排盘</span></template>
        <div class="primary-item">
          <div class="primary-title">{{ latestProfile.nickname }}</div>
          <div class="primary-meta">八字 · 紫微 · 八宅</div>
        </div>
        <el-button type="primary" size="large" @click="router.push(`/profiles/${latestProfile.id}`)">查看排盘</el-button>
      </el-card>

      <div class="actions">
        <router-link to="/profiles"><el-button>我的档案（{{ profiles.list.length }}）</el-button></router-link>
        <el-button @click="router.push('/profiles/new')">新建档案</el-button>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import { useProfilesStore } from '@/stores/profiles'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const router = useRouter()
const chat = useChatStore()
const profiles = useProfilesStore()

const loading = computed(() => chat.loadingSessions || profiles.loading)
const recentSession = computed(() => chat.sessions[0] ?? null)
const latestProfile = computed(() => profiles.list[0] ?? null)

function formatTime(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

onMounted(async () => {
  await Promise.all([chat.loadSessions(), profiles.loadList()])
  // 主流程：无档案的新用户直达建档（输入个人信息），避免多余中转。
  if (!profiles.list.length) {
    await router.replace('/profiles/new')
  }
})
</script>

<style scoped>
.primary { border-radius: 12px; margin-bottom: 20px; }
.card-title { font-weight: 600; }
.primary-item { margin-bottom: 16px; }
.primary-title { font-size: 16px; font-weight: 700; }
.primary-meta { color: var(--liki-text-2); font-size: 13px; margin-top: 4px; }
.actions { display: flex; gap: 12px; }
</style>