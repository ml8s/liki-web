<template>
  <section>
    <AppAlert v-if="chat.error" :title="chat.error" type="error" />
    <AppAlert v-else-if="profiles.error" :title="profiles.error" type="error" />

    <AppSkeleton v-if="loading" :rows="5" />

    <template v-else>
      <!-- Hero：墨纸紫金 · 编辑感首屏 -->
      <div class="hero">
        <div class="hero-gold" />
        <p class="hero-kicker">懂命理，用 Liki</p>
        <h1 class="hero-title">工作台</h1>
        <p class="hero-sub">{{ purchased ? '六大命理服务已就绪，随时对话解读。' : '免费排盘看全，Liki Chat 解锁专业解读。' }}</p>
        <div class="hero-actions">
          <el-button type="primary" size="large" @click="router.push('/chat')">{{ purchased ? '开始 Chat' : '解锁 Liki Chat' }}</el-button>
        </div>
      </div>

      <!-- 服务 bento：发现 + 入口 -->
      <div class="services">
        <div
          v-for="s in chatServices"
          :key="s.key"
          :class="['service-card', s.key === 'bazi' ? 'feature' : '']"
        >
          <span class="sv-glyph font-serif">{{ GLYPH[s.key] || s.label[0] }}</span>
          <div class="sv-copy">
            <div class="sv-name font-serif">{{ s.label }}</div>
            <div class="sv-desc">{{ s.desc }}</div>
          </div>
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
          <router-link to="/profiles" class="more-link">全部档案</router-link>
        </div>
        <div class="recent-profiles">
          <div
            v-for="p in recentProfiles"
            :key="p.id"
            class="recent-profile"
            @click="router.push(`/profiles/${p.id}`)"
          >
            <span class="rp-avatar font-serif">{{ p.nickname.slice(0, 1) }}</span>
            <span class="rp-name">{{ p.nickname }}</span>
            <span class="rp-meta">{{ p.gender === 'male' ? '男' : '女' }}</span>
          </div>
          <div class="recent-profile add" @click="router.push('/profiles/new')">＋ 新建档案</div>
        </div>

        <!-- 最近会话 -->
        <div v-if="recentSessions.length" class="section-head">
          <h2>最近会话</h2>
          <router-link to="/chat" class="more-link">全部会话</router-link>
        </div>
        <div v-if="recentSessions.length" class="recent-sessions">
          <div
            v-for="s in recentSessions"
            :key="s.id"
            class="recent-session"
            @click="router.push(`/chat/${s.id}`)"
          >
            <span class="rs-title">{{ s.profileName || s.title }}</span>
            <span class="rs-meta">{{ skillLabel(s.skill) }} · {{ formatDateTime(s.last_message_at ?? s.updated_at) }}</span>
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
import { formatDateTime } from '@/utils/format'
import AppAlert from '@/components/app/AppAlert.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const GLYPH: Record<string, string> = { bazi: '命', naming: '名', divination: '卦', fengshui: '宅', huangli: '日', bond: '缘' }

const route = useRoute()
const router = useRouter()
const chat = useChatStore()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()

const loading = computed(() => chat.loadingSessions || profiles.loading || entitlements.loading)
const purchased = computed(() => entitlements.allowedSkills.length > 0)
const recentProfiles = computed(() => profiles.list.slice(0, 3))
const recentSessions = computed(() =>
  chat.sessions.slice(0, 3).map(s => ({
    ...s,
    profileName: profiles.list.find(p => p.id === s.profile_id)?.nickname ?? '',
  })),
)

function startService(key: string) {
  const svc = serviceByKey(key)
  // 特殊入口（如合盘 → /synastry）不走对话。
  if (svc?.route) {
    void router.push(svc.route)
    return
  }
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
.hero {
  position: relative;
  overflow: hidden;
  margin-bottom: 28px;
  padding: 34px 30px 30px;
  border-radius: 14px;
  color: #fff;
  background:
    radial-gradient(120% 140% at 85% -20%, rgba(109, 90, 207, 0.55), transparent 60%),
    linear-gradient(135deg, #232733 0%, #2c2740 100%);
}
.hero-gold {
  position: absolute;
  top: 0; left: 30px; right: 30px;
  height: 2px;
  background: linear-gradient(90deg, var(--liki-gold), rgba(201, 164, 94, 0.2));
}
.hero-kicker {
  margin: 0 0 8px;
  font-size: 13px;
  letter-spacing: 0.14em;
  color: var(--liki-gold);
}
.hero-title {
  margin: 0 0 8px;
  font-family: var(--liki-font-serif);
  font-size: 32px;
  font-weight: 700;
  line-height: 1.15;
  letter-spacing: 0.01em;
}
.hero-sub { margin: 0 0 22px; font-size: 15px; color: rgba(255, 255, 255, 0.78); text-wrap: pretty; }
.hero-actions .el-button { --el-button-bg-color: var(--liki-gold); --el-button-border-color: var(--liki-gold); --el-button-hover-bg-color: #d8b877; --el-button-hover-border-color: #d8b877; --el-button-text-color: #3a2f14; --el-button-hover-text-color: #3a2f14; }

.services {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 12px;
  margin-bottom: 28px;
}
.service-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: 1px solid var(--liki-line);
  border-radius: 12px;
  padding: 16px;
  background: var(--liki-surface);
  transition: transform var(--liki-dur) var(--liki-ease), box-shadow var(--liki-dur) var(--liki-ease), border-color var(--liki-dur) var(--liki-ease);
}
.service-card:hover {
  transform: translateY(-2px);
  border-color: var(--liki-primary-border);
  box-shadow: 0 6px 18px rgba(30, 34, 48, 0.08);
}
.service-card.feature { grid-column: span 2; }
.service-card.feature::before {
  content: '';
  position: absolute;
  top: 0; left: 16px; right: 16px;
  height: 2px;
  background: linear-gradient(90deg, var(--liki-gold), transparent);
}
.sv-glyph {
  display: inline-grid;
  place-items: center;
  width: 40px; height: 40px;
  border-radius: 10px;
  background: var(--liki-gold-tint);
  color: #7d6229;
  font-size: 20px;
  font-weight: 600;
}
.sv-copy { display: flex; flex-direction: column; gap: 4px; flex: 1; }
.sv-name { font-size: 18px; font-weight: 700; line-height: 1.2; }
.sv-desc { font-size: 13px; color: var(--liki-text-2); }

.empty-guide {
  border: 1px dashed var(--liki-primary-border); border-radius: 12px;
  padding: 24px; text-align: center; background: var(--liki-surface);
  margin-bottom: 20px;
}
.empty-guide p { margin: 0 0 14px; color: var(--liki-text-2); }

.section-head { display: flex; align-items: baseline; justify-content: space-between; margin: 20px 0 10px; }
.section-head h2 { margin: 0; font-size: 16px; font-weight: 700; letter-spacing: -0.01em; }
.more-link { font-size: 13px; color: var(--liki-primary); text-decoration: none; }

.recent-profiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; }
.recent-profile {
  display: flex; align-items: center; gap: 10px;
  border: 1px solid var(--liki-line); border-radius: 12px;
  padding: 12px 14px; cursor: pointer; background: var(--liki-surface);
  transition: border-color var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease);
}
.recent-profile:hover { border-color: var(--liki-primary-border); transform: translateY(-1px); }
.recent-profile.add { border-style: dashed; color: var(--liki-primary); justify-content: center; }
.rp-avatar {
  display: inline-grid; place-items: center;
  width: 34px; height: 34px; border-radius: 50%;
  background: var(--liki-primary-tint); color: var(--liki-primary);
  font-size: 16px; font-weight: 600;
}
.rp-name { font-weight: 600; flex: 1; }
.rp-meta { color: var(--liki-text-3); font-size: 12px; }

.recent-sessions { display: grid; gap: 8px; }
.recent-session {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  border: 1px solid var(--liki-line); border-radius: 12px;
  padding: 13px 14px; cursor: pointer; background: var(--liki-surface);
  transition: border-color var(--liki-dur) var(--liki-ease);
}
.recent-session:hover { border-color: var(--liki-primary-border); }
.rs-title { font-weight: 600; }
.rs-meta { color: var(--liki-text-2); font-size: 12px; white-space: nowrap; }

@media (max-width: 900px) {
  .services { grid-template-columns: repeat(2, 1fr); }
  .service-card.feature { grid-column: span 2; }
}
@media (max-width: 520px) {
  .services { grid-template-columns: 1fr; }
  .service-card.feature { grid-column: span 1; }
  .hero { padding: 26px 22px 24px; }
  .hero-title { font-size: 27px; }
}
</style>