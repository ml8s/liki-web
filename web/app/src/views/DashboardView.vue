<template>
  <section>
    <el-alert
      v-if="chat.error"
      :title="chat.error"
      type="error"
      :closable="false"
      show-icon
    />
    <el-alert
      v-else-if="profiles.error"
      :title="profiles.error"
      type="error"
      :closable="false"
      show-icon
    />
    <div
      v-if="entitlements.error"
      class="entitlements-error"
    >
      <el-alert
        :title="entitlements.error"
        type="error"
        :closable="false"
        show-icon
      />
      <el-button
        size="small"
        @click="entitlements.load(true)"
      >
        {{ $t('common.retry') }}
      </el-button>
    </div>

    <el-skeleton
      v-if="loading"
      :rows="5"
      animated
      aria-busy="true"
    />

    <template v-else>
      <div class="hero">
        <div class="hero-accent" />
        <p class="hero-kicker">
          {{ $t('auth.slogan') }}
        </p>
        <h1 class="hero-title">
          {{ $t('router.dashboard') }}
        </h1>
        <p class="hero-sub">
          {{ heroSubtitle }}
        </p>
      </div>

      <el-card
        v-if="dashboardState === 'no-profile'"
        class="app-card app-card--highlight primary-guide"
        shadow="never"
      >
        <div class="primary-body">
          <div class="primary-copy">
            <h2 class="app-card__title">
              {{ $t('dashboard.noBirthProfile.title') }}
            </h2>
            <p>{{ $t('dashboard.noBirthProfile.body') }}</p>
          </div>
          <div class="primary-actions">
            <el-button
              type="primary"
              size="large"
              @click="router.push('/birth-profiles/new')"
            >
              {{ $t('dashboard.noBirthProfile.cta') }}
            </el-button>
          </div>
        </div>
      </el-card>

      <template v-else>
        <el-card
          class="app-card app-card--highlight"
          shadow="never"
        >
          <div class="primary-body">
            <div class="primary-copy">
              <template v-if="dashboardState === 'free-profile'">
                <h2 class="app-card__title">
                  {{ $t('dashboard.freeBirthProfile.title') }}
                </h2>
                <p v-if="primaryBirthProfile">
                  {{ primaryBirthProfile.nickname }} · {{ birthSummary(primaryBirthProfile) }}
                </p>
                <p class="primary-note">
                  {{ $t('dashboard.freeBirthProfile.note') }}
                </p>
              </template>

              <template v-else-if="dashboardState === 'paid-no-session'">
                <h2 class="app-card__title">
                  {{ $t('dashboard.paidNoSession.title') }}
                </h2>
                <p>{{ $t('dashboard.paidNoSession.body') }}</p>
              </template>

              <template v-else-if="latestSession">
                <h2 class="app-card__title">
                  {{ $t('dashboard.continue.title') }}
                </h2>
                <p>
                  {{ latestSessionBirthProfileName || latestSession.title }} ·
                  {{ formatDateTime(latestSession.last_message_at ?? latestSession.updated_at) }}
                </p>
              </template>
            </div>

            <div class="primary-actions">
              <el-button
                v-if="dashboardState === 'free-profile'"
                size="large"
                @click="viewChart"
              >
                {{ $t('dashboard.freeBirthProfile.viewChart') }}
              </el-button>
              <el-button
                v-if="dashboardState === 'free-profile'"
                type="primary"
                size="large"
                @click="startInterpretation"
              >
                {{ $t('dashboard.freeBirthProfile.start') }}
              </el-button>

              <el-button
                v-if="dashboardState === 'paid-no-session'"
                type="primary"
                size="large"
                @click="startInterpretation"
              >
                {{ $t('dashboard.paidNoSession.cta') }}
              </el-button>

              <el-button
                v-if="dashboardState === 'paid-continue' && latestSession"
                type="primary"
                size="large"
                @click="router.push(`/chat/${latestSession.id}`)"
              >
                {{ $t('dashboard.continue.cta') }}
              </el-button>
            </div>
          </div>
        </el-card>

        <el-card
          class="app-card"
          shadow="never"
        >
          <template #header>
            <div class="app-card__header">
              <h2 class="app-card__title">
                {{ $t('dashboard.recentBirthProfiles') }}
              </h2>
              <router-link
                to="/birth-profiles"
                class="more-link"
              >
                {{ $t('dashboard.allBirthProfiles') }}
              </router-link>
            </div>
          </template>
          <div class="recent-profiles">
            <router-link
              v-for="profile in recentBirthProfiles"
              :key="profile.id"
              class="recent-profile"
              :to="`/birth-profiles/${profile.id}`"
            >
              <span class="rp-avatar font-serif">{{ profile.nickname.slice(0, 1) }}</span>
              <span class="rp-name">{{ profile.nickname }}</span>
              <span class="rp-meta">{{ $t(profile.gender === 'male' ? 'profiles.male' : 'profiles.female') }}</span>
            </router-link>
            <router-link
              class="recent-profile add"
              to="/birth-profiles/new"
            >
              ＋ {{ $t('profiles.create') }}
            </router-link>
          </div>
        </el-card>

        <el-card
          v-if="recentSessions.length"
          class="app-card"
          shadow="never"
        >
          <template #header>
            <div class="app-card__header">
              <h2 class="app-card__title">
                {{ $t('dashboard.recentSessions') }}
              </h2>
              <router-link
                to="/chat"
                class="more-link"
              >
                {{ $t('dashboard.allSessions') }}
              </router-link>
            </div>
          </template>
          <div class="recent-sessions">
            <router-link
              v-for="session in recentSessions"
              :key="session.id"
              class="recent-session"
              :to="`/chat/${session.id}`"
            >
              <span class="rs-title">{{ session.profileName || session.title }}</span>
              <span class="rs-meta">
                {{ formatDateTime(session.last_message_at ?? session.updated_at) }}
              </span>
            </router-link>
          </div>
        </el-card>
      </template>

      <!-- 场景/服务入口与档案状态解耦：无档案用户同样可见，点击走 startService 的建档流程 -->
      <el-card
        class="app-card"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('dashboard.moreScenarios') }}
          </h2>
        </template>
        <ServiceEntryList
          :items="scenarioEntries"
          @select="startService"
        />
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useEntitlementsStore } from '@/stores/entitlements'
import { birthSummary, type BirthProfile } from '@/api/birth-profiles'
import { resolveDashboardState, type DashboardState } from '@/composables/dashboardState'
import { normalizeScenario, scenarioService } from '@/composables/scenario'
import { chatUrl, scenarioById, scenarioEntryItems } from '@/config/chat'
import { useI18n } from 'vue-i18n'
import { formatDateTime } from '@/utils/format'
import ServiceEntryList from '@/components/app/ServiceEntryList.vue'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const chat = useChatStore()
const profiles = useBirthProfilesStore()
const entitlements = useEntitlementsStore()
const drafts = useFormDraftsStore()

const loading = computed(() =>
  chat.loadingSessions || profiles.loading || entitlements.loading,
)

const scenarioEntries = computed(() => scenarioEntryItems(t))
const hasBirthProfile = computed(() => profiles.list.length > 0)

function startScenario(scenario: string) {
  const service = scenarioService(normalizeScenario(scenario))
  if (service) {
    startService(service)
    return
  }
  void router.push('/chat')
}
const hasEntitlement = computed(() => entitlements.purchased)
const primaryBirthProfile = computed<BirthProfile | undefined>(() => profiles.list[0])
const latestSession = computed(() => chat.sessions[0])
const latestSessionBirthProfileName = computed(() =>
  latestSession.value
    ? profiles.list.find(item => item.id === latestSession.value?.profile_id)?.nickname ?? ''
    : '',
)
const recentBirthProfiles = computed(() => profiles.list.slice(0, 3))
const recentSessions = computed(() =>
  chat.sessions.slice(0, 3).map(session => ({
    ...session,
    profileName: profiles.list.find(item => item.id === session.profile_id)?.nickname ?? '',
  })),
)

const dashboardState = computed<DashboardState>(() => resolveDashboardState({
  hasBirthProfile: hasBirthProfile.value,
  hasEntitlement: hasEntitlement.value,
  hasSession: Boolean(latestSession.value),
}))

// Record 查找天然穷尽 DashboardState 四态，无漏 default 分支的隐患。
const heroSubtitle = computed(
  () =>
    ({
      'no-profile': t('dashboard.state.noBirthProfile'),
      'free-profile': t('dashboard.state.freeBirthProfile'),
      'paid-no-session': t('dashboard.state.paidNoSession'),
      'paid-continue': t('dashboard.state.paidContinue'),
    })[dashboardState.value],
)

function viewChart() {
  if (!primaryBirthProfile.value) return
  void router.push(`/birth-profiles/${primaryBirthProfile.value.id}`)
}

function startInterpretation() {
  if (!primaryBirthProfile.value) return
  void router.push(chatUrl(primaryBirthProfile.value.id, 'bazi'))
}

function startService(key: string) {
  drafts.set('service', key)
  const scenario = scenarioById(key)
  if (scenario?.route) {
    void router.push(scenario.route)
    return
  }
  if (!profiles.list.length) {
    void router.push('/birth-profiles/new')
    return
  }
  const profile = profiles.list[0]
  void router.push(chatUrl(profile.id, key))
}


onMounted(async () => {
  await Promise.allSettled([
    profiles.loadList(),
    entitlements.load(),
    chat.loadSessions(),
  ])

  const rawScenario = route.query.scenario
  const scenario = Array.isArray(rawScenario) ? String(rawScenario[0] ?? '') : String(rawScenario ?? '')
  if (normalizeScenario(scenario) !== 'general') {
    const query = { ...route.query }
    delete query.scenario
    void router.replace({ query })
    startScenario(scenario)
    return
  }

  const raw = route.query.service
  const key = Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '')
  if (key && scenarioById(key)) {
    void router.replace({ query: {} })
    startService(key)
  }
})
</script>

<style scoped>
/* entitlements 加载失败为非阻塞错误：有数据仍可用，仅提示并可重试刷新面板 */
.entitlements-error {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
}
.entitlements-error :deep(.el-alert) {
  flex: 1;
}
.hero {
  --hero-pad-x: 30px;
  position: relative;
  overflow: hidden;
  margin-bottom: 24px;
  padding: 34px var(--hero-pad-x) 30px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  color: var(--liki-ink);
  background:
    radial-gradient(90% 120% at 88% -20%, var(--liki-primary-alpha-10), transparent 62%),
    linear-gradient(180deg, var(--liki-surface) 0%, var(--liki-paper) 100%);
  box-shadow: var(--shadow-card);
}
.hero-accent {
  position: absolute;
  top: 0;
  left: var(--hero-pad-x);
  right: var(--hero-pad-x);
  height: 2px;
  background: linear-gradient(90deg, var(--liki-primary), var(--liki-primary-alpha-12));
}
.hero-kicker {
  margin: 0 0 8px;
  font-size: var(--font-size-label);
  letter-spacing: 0.14em;
  color: var(--liki-primary);
  font-weight: 600;
}
.hero-title {
  margin: 0 0 8px;
  font-size: clamp(var(--font-size-display-sm), 4vw, var(--font-size-display));
  font-weight: 650;
  line-height: 1.15;
  letter-spacing: -0.035em;
}
.hero-sub {
  margin: 0;
  font-size: var(--font-size-body-lg);
  color: var(--liki-text-2);
  text-wrap: pretty;
}

/* 主状态卡内容布局：容器用统一 app-card（强调态 = app-card--highlight）。 */
.primary-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}
/* 无档案引导态：dashed 边框提示「需要动作」（仅此一处状态变体）。 */
.primary-guide {
  border-style: dashed;
  background: linear-gradient(180deg, var(--liki-surface) 0%, var(--liki-paper) 100%);
}
.primary-copy {
  min-width: 0;
}
.primary-copy p {
  margin: 0 0 4px;
  color: var(--liki-text-2);
  font-size: var(--font-size-body);
  font-variant-numeric: tabular-nums;
}
.primary-note {
  color: var(--liki-text-3) !important;
  font-size: var(--font-size-label);
}
.primary-actions {
  flex-shrink: 0;
  display: flex;
  gap: 10px;
}


.more-link {
  font-size: var(--font-size-label);
  color: var(--liki-primary);
  text-decoration: none;
}

.recent-profiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 10px;
}
.recent-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  padding: 12px 14px;
  cursor: pointer;
  background: var(--liki-surface);
  color: inherit;
  text-decoration: none;
  transition:
    border-color var(--liki-dur) var(--liki-ease),
    background var(--liki-dur) var(--liki-ease);
}
.recent-profile:hover {
  border-color: var(--liki-primary-border);
  background: var(--liki-paper);
}
.recent-profile.add {
  border-style: dashed;
  color: var(--liki-primary);
  justify-content: center;
}
.rp-avatar {
  display: inline-grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-size: var(--font-size-title-sm);
  font-weight: 600;
}
.rp-name {
  flex: 1;
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rp-meta {
  flex-shrink: 0;
  color: var(--liki-text-3);
  font-size: var(--font-size-caption);
}

.recent-sessions {
  display: grid;
  gap: 8px;
}
.recent-session {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  padding: 13px 14px;
  cursor: pointer;
  background: var(--liki-surface);
  color: inherit;
  text-decoration: none;
  transition:
    border-color var(--liki-dur) var(--liki-ease),
    background var(--liki-dur) var(--liki-ease);
}
.recent-session:hover {
  border-color: var(--liki-primary-border);
  background: var(--liki-paper);
}
.rs-title {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rs-meta {
  flex-shrink: 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .primary-body {
    align-items: stretch;
    flex-direction: column;
    gap: 16px;
  }
  .primary-actions {
    width: 100%;
    flex-direction: column;
  }
  .primary-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
@media (max-width: 480px) {
  .hero {
    --hero-pad-x: 22px;
    padding: 26px var(--hero-pad-x) 24px;
  }
}
</style>
