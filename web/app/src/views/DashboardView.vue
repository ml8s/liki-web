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

      <div
        v-if="dashboardState === 'no-profile'"
        class="primary-card guide"
      >
        <div class="primary-copy">
          <h2>{{ $t('dashboard.noProfile.title') }}</h2>
          <p>{{ $t('dashboard.noProfile.body') }}</p>
        </div>
        <el-button
          type="primary"
          size="large"
          @click="router.push('/profiles/new')"
        >
          {{ $t('dashboard.noProfile.cta') }}
        </el-button>
      </div>

      <template v-else>
        <div class="primary-card">
          <div class="primary-copy">
            <template v-if="dashboardState === 'free-profile'">
              <h2>{{ $t('dashboard.freeProfile.title') }}</h2>
              <p v-if="primaryProfile">
                {{ primaryProfile.nickname }} · {{ birthSummary(primaryProfile) }}
              </p>
              <p class="primary-note">
                {{ $t('dashboard.freeProfile.note') }}
              </p>
            </template>

            <template v-else-if="dashboardState === 'paid-no-session'">
              <h2>{{ $t('dashboard.paidNoSession.title') }}</h2>
              <p>{{ $t('dashboard.paidNoSession.body') }}</p>
            </template>

            <template v-else-if="latestSession">
              <h2>{{ $t('dashboard.continue.title') }}</h2>
              <p>
                {{ latestSessionProfileName || latestSession.title }} ·
                {{ skillLabel(latestSession.skill) }} ·
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
              {{ $t('dashboard.freeProfile.viewChart') }}
            </el-button>
            <el-button
              v-if="dashboardState === 'free-profile'"
              type="primary"
              size="large"
              @click="startInterpretation"
            >
              {{ $t('dashboard.freeProfile.start') }}
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

        <div class="section-head">
          <h2>{{ $t('dashboard.recentProfiles') }}</h2>
          <router-link
            to="/profiles"
            class="more-link"
          >
            {{ $t('dashboard.allProfiles') }}
          </router-link>
        </div>
        <div class="recent-profiles">
          <router-link
            v-for="profile in recentProfiles"
            :key="profile.id"
            class="recent-profile"
            :to="`/profiles/${profile.id}`"
          >
            <span class="rp-avatar font-serif">{{ profile.nickname.slice(0, 1) }}</span>
            <span class="rp-name">{{ profile.nickname }}</span>
            <span class="rp-meta">{{ $t(profile.gender === 'male' ? 'profiles.male' : 'profiles.female') }}</span>
          </router-link>
          <router-link
            class="recent-profile add"
            to="/profiles/new"
          >
            ＋ {{ $t('profiles.create') }}
          </router-link>
        </div>

        <div
          v-if="recentSessions.length"
          class="section-head"
        >
          <h2>{{ $t('dashboard.recentSessions') }}</h2>
          <router-link
            to="/chat"
            class="more-link"
          >
            {{ $t('dashboard.allSessions') }}
          </router-link>
        </div>
        <div
          v-if="recentSessions.length"
          class="recent-sessions"
        >
          <router-link
            v-for="session in recentSessions"
            :key="session.id"
            class="recent-session"
            :to="`/chat/${session.id}`"
          >
            <span class="rs-title">{{ session.profileName || session.title }}</span>
            <span class="rs-meta">
              {{ skillLabel(session.skill) }} ·
              {{ formatDateTime(session.last_message_at ?? session.updated_at) }}
            </span>
          </router-link>
        </div>

        <div class="section-head scenarios-head">
          <h2>{{ $t('dashboard.moreScenarios') }}</h2>
        </div>
        <div class="services">
          <div
            v-for="service in chatServices"
            :key="service.key"
            :class="['service-card', service.key === 'bazi' ? 'feature' : '']"
          >
            <span class="sv-glyph font-serif">{{ $t(service.labelKey).slice(0, 1) }}</span>
            <div class="sv-copy">
              <div class="sv-name font-serif">
                {{ $t(service.labelKey) }}
              </div>
              <div class="sv-desc">
                {{ $t(service.descKey) }}
              </div>
            </div>
            <el-button
              type="primary"
              size="small"
              plain
              @click="startService(service.key)"
            >
              {{ $t(service.ctaKey) }}
            </el-button>
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
import { birthSummary, type Profile } from '@/api/profiles'
import { resolveDashboardState, type DashboardState } from '@/composables/dashboardState'
import { normalizeScenario, scenarioService } from '@/composables/scenario'
import { chatServices, chatUrl, serviceByKey } from '@/config/chat'
import { useI18n } from 'vue-i18n'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const chat = useChatStore()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()

const loading = computed(() =>
  chat.loadingSessions || profiles.loading || entitlements.loading,
)
const hasProfile = computed(() => profiles.list.length > 0)

function startScenario(scenario: string) {
  const service = scenarioService(normalizeScenario(scenario))
  if (service) {
    startService(service)
    return
  }
  void router.push('/chat')
}
const hasEntitlement = computed(() => entitlements.purchased)
const primaryProfile = computed<Profile | undefined>(() => profiles.list[0])
const latestSession = computed(() => chat.sessions[0])
const latestSessionProfileName = computed(() =>
  latestSession.value
    ? profiles.list.find(item => item.id === latestSession.value?.profile_id)?.nickname ?? ''
    : '',
)
const recentProfiles = computed(() => profiles.list.slice(0, 3))
const recentSessions = computed(() =>
  chat.sessions.slice(0, 3).map(session => ({
    ...session,
    profileName: profiles.list.find(item => item.id === session.profile_id)?.nickname ?? '',
  })),
)

const dashboardState = computed<DashboardState>(() => resolveDashboardState({
  hasProfile: hasProfile.value,
  hasEntitlement: hasEntitlement.value,
  hasSession: Boolean(latestSession.value),
}))

// Record 查找天然穷尽 DashboardState 四态，无漏 default 分支的隐患。
const heroSubtitle = computed(
  () =>
    ({
      'no-profile': t('dashboard.state.noProfile'),
      'free-profile': t('dashboard.state.freeProfile'),
      'paid-no-session': t('dashboard.state.paidNoSession'),
      'paid-continue': t('dashboard.state.paidContinue'),
    })[dashboardState.value],
)

function viewChart() {
  if (!primaryProfile.value) return
  void router.push(`/profiles/${primaryProfile.value.id}`)
}

function startInterpretation() {
  if (!primaryProfile.value) return
  void router.push(chatUrl(primaryProfile.value.id, 'bazi'))
}

function startService(key: string) {
  const service = serviceByKey(key)
  if (service?.route) {
    void router.push(service.route)
    return
  }
  if (!profiles.list.length) {
    try {
      sessionStorage.setItem('liki-service', key)
    } catch {
      // sessionStorage 不可用时不阻断主流程。
    }
    void router.push('/profiles/new')
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
  if (key && serviceByKey(key)) {
    void router.replace({ query: {} })
    startService(key)
  }
})
</script>

<style scoped>
.hero {
  position: relative;
  overflow: hidden;
  margin-bottom: 24px;
  padding: 34px 30px 30px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  color: var(--liki-ink);
  background:
    radial-gradient(90% 120% at 88% -20%, rgba(83, 58, 253, 0.10), transparent 62%),
    linear-gradient(180deg, #ffffff 0%, var(--liki-paper) 100%);
  box-shadow: var(--shadow-card);
}
.hero-accent {
  position: absolute;
  top: 0;
  left: 30px;
  right: 30px;
  height: 2px;
  background: linear-gradient(90deg, var(--liki-primary), rgba(83, 58, 253, 0.12));
}
.hero-kicker {
  margin: 0 0 8px;
  font-size: 13px;
  letter-spacing: 0.14em;
  color: var(--liki-primary);
  font-weight: 600;
}
.hero-title {
  margin: 0 0 8px;
  font-size: 32px;
  font-weight: 650;
  line-height: 1.15;
  letter-spacing: -0.035em;
}
.hero-sub {
  margin: 0;
  font-size: 15px;
  color: var(--liki-text-2);
  text-wrap: pretty;
}

.primary-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;
  padding: 24px 26px;
  border: 1px solid var(--liki-primary-border);
  border-radius: var(--radius-card);
  background: var(--liki-surface);
  box-shadow: var(--shadow-card);
}
.primary-card.guide {
  border-style: dashed;
  background: linear-gradient(180deg, #ffffff 0%, var(--liki-paper) 100%);
}
.primary-copy {
  min-width: 0;
}
.primary-copy h2 {
  margin: 0 0 8px;
  font-family: var(--liki-font-serif);
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
}
.primary-copy p {
  margin: 0 0 4px;
  color: var(--liki-text-2);
  font-size: 14px;
}
.primary-note {
  color: var(--liki-text-3) !important;
  font-size: 13px;
}
.primary-actions {
  flex-shrink: 0;
  display: flex;
  gap: 10px;
}

.services {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 14px;
}
.service-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-card);
  padding: 16px;
  background: var(--liki-surface);
  transition:
    transform var(--liki-dur) var(--liki-ease),
    box-shadow var(--liki-dur) var(--liki-ease),
    border-color var(--liki-dur) var(--liki-ease);
}
.service-card:hover {
  transform: translateY(-2px);
  border-color: var(--liki-primary-border);
  box-shadow: 0 6px 18px rgba(30, 34, 48, 0.08);
}
.service-card.feature {
  grid-column: span 2;
}
.service-card.feature::before {
  content: '';
  position: absolute;
  top: 0;
  left: 16px;
  right: 16px;
  height: 2px;
  background: linear-gradient(90deg, var(--liki-primary), transparent);
}
.sv-glyph {
  display: inline-grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--liki-primary-tint);
  color: var(--liki-primary);
  font-size: 20px;
  font-weight: 600;
}
.sv-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}
.sv-name {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
}
.sv-desc {
  font-size: 13px;
  color: var(--liki-text-2);
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 20px 0 10px;
}
.section-head h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.scenarios-head {
  margin-top: 28px;
}
.more-link {
  font-size: 13px;
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
  border-radius: 12px;
  padding: 12px 14px;
  cursor: pointer;
  background: var(--liki-surface);
  color: inherit;
  text-decoration: none;
  transition:
    border-color var(--liki-dur) var(--liki-ease),
    transform var(--liki-dur) var(--liki-ease);
}
.recent-profile:hover {
  border-color: var(--liki-primary-border);
  transform: translateY(-1px);
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
  font-size: 16px;
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
  font-size: 12px;
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
  border-radius: 12px;
  padding: 13px 14px;
  cursor: pointer;
  background: var(--liki-surface);
  color: inherit;
  text-decoration: none;
  transition: border-color var(--liki-dur) var(--liki-ease);
}
.recent-session:hover {
  border-color: var(--liki-primary-border);
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
  font-size: 12px;
  white-space: nowrap;
}

@media (max-width: 760px) {
  .primary-card {
    align-items: stretch;
    flex-direction: column;
    gap: 16px;
    padding: 22px;
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
@media (max-width: 900px) {
  .services {
    grid-template-columns: repeat(2, 1fr);
  }
  .service-card.feature {
    grid-column: span 2;
  }
}
@media (max-width: 520px) {
  .services {
    grid-template-columns: 1fr;
  }
  .service-card.feature {
    grid-column: span 1;
  }
  .hero {
    padding: 26px 22px 24px;
  }
  .hero-title {
    font-size: 27px;
  }
}
</style>
