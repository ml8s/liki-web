<template>
  <section>
    <AppPageHeader title="工作台" subtitle="继续上次咨询，或管理你的档案。" />

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert v-else-if="profiles.error" :title="profiles.error" type="error" />

    <div v-if="!loading && !profiles.list.length" class="onboarding">
      <h2 class="ob-title font-serif">欢迎来到 Liki，四步开始</h2>
      <div class="ob-steps">
        <div class="ob-step" :class="{ done: profiles.list.length > 0 }">
          <span class="ob-n">1</span>
          <span class="ob-t">建立档案</span>
          <span class="ob-state">{{ profiles.list.length ? '已完成' : '进行中' }}</span>
        </div>
        <div class="ob-step" :class="{ ready: profiles.list.length > 0 }">
          <span class="ob-n">2</span>
          <span class="ob-t">免费看盘</span>
          <span class="ob-state">建档后可看八字·紫微·八宅</span>
        </div>
        <div class="ob-step">
          <span class="ob-n">3</span>
          <span class="ob-t">解锁对话</span>
          <span class="ob-state">付费解锁专业解读</span>
        </div>
      </div>
      <el-button type="primary" size="large" @click="router.push('/profiles/new')">创建第一个档案</el-button>
    </div>

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
.onboarding {
  background: var(--liki-surface); border: 1px solid var(--liki-line);
  border-radius: 12px; padding: 24px; margin-bottom: 20px;
}
.ob-title { margin: 0 0 16px; font-size: 18px; }
.ob-steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 18px; }
.ob-step { border: 1px solid var(--liki-line); border-radius: 8px; padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; }
.ob-step.done { border-color: var(--el-color-success); }
.ob-step.ready { border-color: var(--liki-primary-border); }
.ob-n { display: inline-grid; place-items: center; width: 24px; height: 24px; border-radius: 50%; background: var(--liki-primary); color: #fff; font-size: 13px; }
.ob-t { font-weight: 600; }
.ob-state { color: var(--liki-text-2); font-size: 12px; }
</style>