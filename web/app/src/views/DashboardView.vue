<template>
  <section>
    <AppPageHeader title="工作台" subtitle="继续上次咨询，或管理你的档案。" />

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert v-else-if="profiles.error" :title="profiles.error" type="error" />
    <AppAlert
      v-if="!loading && !profiles.list.length"
      title="欢迎来到 Liki"
      description="先创建一个档案，免费查看命盘；解锁 Liki Chat 后可自由对话。"
      type="info"
    />

    <AppSkeleton v-if="loading" :rows="4" />

    <template v-else>
      <el-card class="resume" shadow="never">
        <template #header><span class="card-title">继续上次咨询</span></template>
        <template v-if="recentSession">
          <div class="resume-item">
            <div class="resume-title">{{ recentSession.title }}</div>
            <div class="resume-meta">{{ skillLabel(recentSession.skill) }} · {{ formatTime(recentSession.last_message_at ?? recentSession.updated_at) }}</div>
          </div>
          <el-button type="primary" size="large" @click="router.push(`/chat/${recentSession.id}`)">继续</el-button>
        </template>
        <template v-else>
          <p class="hint">还没有会话。先建一个档案，再开启解读吧。</p>
          <el-button type="primary" size="large" @click="router.push('/profiles/new')">新建档案</el-button>
        </template>
      </el-card>

      <div class="quick">
        <router-link to="/profiles">
          <el-button>我的档案{{ profiles.list.length ? `（${profiles.list.length}）` : '' }}</el-button>
        </router-link>
        <router-link to="/chat">
          <el-button>进入 Chat</el-button>
        </router-link>
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

function formatTime(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

onMounted(async () => {
  await Promise.all([chat.loadSessions(), profiles.loadList()])
})
</script>

<style scoped>
.resume { border-radius: 12px; margin-bottom: 20px; }
.card-title { font-weight: 600; }
.resume-item { margin-bottom: 16px; }
.resume-title { font-size: 16px; font-weight: 700; }
.resume-meta { color: var(--liki-text-2); font-size: 13px; margin-top: 4px; }
.hint { color: var(--liki-text-2); margin: 0 0 16px; }
.quick { display: flex; gap: 12px; }
</style>