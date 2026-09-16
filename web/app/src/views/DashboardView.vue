<template>
  <section>
    <AppPageHeader title="工作台" subtitle="选择一项服务，或继续上次的咨询。" />

    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert v-else-if="profiles.error" :title="profiles.error" type="error" />

    <AppSkeleton v-if="loading" :rows="5" />

    <template v-else>
      <!-- 权益状态条 -->
      <div v-if="entitlements.loaded" :class="['status-bar', purchased ? 'on' : '']">
        <span class="status-text">
          {{ purchased ? 'Liki Chat 已开通' : 'Liki Chat 未解锁，解锁后可专业解读' }}
        </span>
        <el-button v-if="!purchased" type="primary" size="small" @click="router.push('/chat')">解锁</el-button>
      </div>

      <!-- 四大服务：发现 + 入口 -->
      <div class="services">
        <div
          v-for="s in chatServices"
          :key="s.key"
          class="service-card"
        >
          <div class="sv-name font-serif">{{ s.label }}</div>
          <div class="sv-desc">{{ s.desc }}</div>
          <el-button type="primary" size="small" plain @click="startService(s.key)">开始</el-button>
        </div>
      </div>

      <!-- 0 档案：建档引导 -->
      <div v-if="!profiles.list.length" class="empty-guide">
        <p>建立档案后，免费查看八字、紫微、八宅三盘。</p>
        <el-button type="primary" size="large" @click="router.push('/profiles/new')">建立档案，免费排盘</el-button>
      </div>

      <template v-else>
        <!-- 最近档案 -->
        <div class="section-head">
          <h2>我的档案</h2>
          <router-link to="/profiles" class="more-link">全部档案 →</router-link>
        </div>
        <div class="recent-profiles">
          <div
            v-for="p in recentProfiles"
            :key="p.id"
            class="recent-profile"
            @click="router.push(`/profiles/${p.id}`)"
          >
            <span class="rp-name font-serif">{{ p.nickname }}</span>
            <span class="rp-meta">{{ p.gender === 'male' ? '男' : '女' }}</span>
          </div>
          <div class="recent-profile add" @click="router.push('/profiles/new')">＋ 新建档案</div>
        </div>

        <!-- 最近会话 -->
        <div v-if="recentSessions.length" class="section-head">
          <h2>最近会话</h2>
          <router-link to="/chat" class="more-link">全部会话 →</router-link>
        </div>
        <div v-if="recentSessions.length" class="recent-sessions">
          <div
            v-for="s in recentSessions"
            :key="s.id"
            class="recent-session"
            @click="router.push(`/chat/${s.id}`)"
          >
            <span class="rs-title">{{ s.title }}</span>
            <span class="rs-meta">{{ skillLabel(s.skill) }} · {{ formatTime(s.last_message_at ?? s.updated_at) }}</span>
          </div>
        </div>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import { useProfilesStore } from '@/stores/profiles'
import { useEntitlementsStore } from '@/stores/entitlements'
import { chatServices, serviceByKey } from '@/config/chat'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()

const loading = computed(() => chat.loadingSessions || profiles.loading || entitlements.loading)
const purchased = computed(() => entitlements.allowedSkills.length > 0)
const recentProfiles = computed(() => profiles.list.slice(0, 3))
const recentSessions = computed(() => chat.sessions.slice(0, 3))

function formatTime(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

function startService(key: string) {
  if (!profiles.list.length) {
    // 无档案：先建档，intent 带到档案详情解读卡预选。
    try { sessionStorage.setItem('liki-service', key) } catch { /* ignore */ }
    void router.push('/profiles/new')
    return
  }
  void router.push(`/chat?profile=${profiles.list[0].id}&service=${key}`)
}

onMounted(async () => {
  await Promise.all([chat.loadSessions(), profiles.loadList(), entitlements.load()])
  // 营销站/外链服务意图：/app?service=<key> → 自动启动该服务。
  const raw = route.query.service
  const key = Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '')
  if (key && serviceByKey(key)) {
    await router.replace({ query: {} })
    startService(key)
  }
})
</script>

<style scoped>
.status-bar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; margin-bottom: 20px;
  border: 1px solid var(--liki-gold-border); border-radius: 10px;
  background: var(--liki-gold-tint); color: #8a6a2f;
}
.status-bar.on { border-color: #bfe6bf; background: #f4faf4; color: #3f7a2d; }
.status-text { font-size: 14px; }

.services {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px;
  margin-bottom: 24px;
}
.service-card {
  display: flex; flex-direction: column; gap: 8px;
  border: 1px solid var(--liki-line); border-radius: 12px;
  padding: 16px; background: var(--liki-surface);
}
.sv-name { font-size: 18px; font-weight: 700; }
.sv-desc { font-size: 13px; color: var(--liki-text-2); min-height: 36px; }

.empty-guide {
  border: 1px dashed var(--liki-primary-border); border-radius: 12px;
  padding: 24px; text-align: center; background: var(--liki-surface);
  margin-bottom: 20px;
}
.empty-guide p { margin: 0 0 14px; color: var(--liki-text-2); }

.section-head { display: flex; align-items: baseline; justify-content: space-between; margin: 20px 0 10px; }
.section-head h2 { margin: 0; font-size: 16px; }
.more-link { font-size: 13px; color: var(--liki-primary); text-decoration: none; }

.recent-profiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; }
.recent-profile {
  display: flex; align-items: center; justify-content: space-between;
  border: 1px solid var(--liki-line); border-radius: 10px;
  padding: 12px 14px; cursor: pointer; background: var(--liki-surface);
  transition: border-color var(--liki-dur) var(--liki-ease);
}
.recent-profile:hover { border-color: var(--liki-primary-border); }
.recent-profile.add { border-style: dashed; color: var(--liki-primary); }
.rp-name { font-weight: 600; }
.rp-meta { color: var(--liki-text-3); font-size: 12px; }

.recent-sessions { display: grid; gap: 8px; }
.recent-session {
  display: flex; align-items: center; justify-content: space-between;
  border: 1px solid var(--liki-line); border-radius: 10px;
  padding: 12px 14px; cursor: pointer; background: var(--liki-surface);
}
.recent-session:hover { border-color: var(--liki-primary-border); }
.rs-title { font-weight: 600; }
.rs-meta { color: var(--liki-text-2); font-size: 12px; }

@media (max-width: 900px) {
  .services { grid-template-columns: repeat(2, 1fr); }
}
</style>