<template>
  <section>
    <AppPageHeader
      :title="$t('router.account')"
      :subtitle="$t('account.subtitle')"
    >
      <template #actions>
        <el-button
          tag="router-link"
          to="/orders"
        >
          {{ $t('nav.orders') }}
        </el-button>
        <el-button
          tag="router-link"
          to="/partners"
        >
          {{ $t('nav.partner') }}
        </el-button>
        <el-button
          :loading="revoking"
          type="danger"
          plain
          @click="logout"
        >
          {{ $t('account.logoutAll') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('account.info.title') }}
        </h2>
      </template>
      <el-descriptions
        :column="1"
        border
      >
        <el-descriptions-item :label="$t('auth.email')">
          {{ auth.user?.email ?? '—' }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('account.info.status')">
          <el-tag
            :type="auth.user?.status === 'active' ? 'success' : 'warning'"
            size="small"
          >
            {{ statusLabel(auth.user?.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item
          v-if="auth.user?.last_login_at"
          :label="$t('account.info.lastLogin')"
        >
          {{ formatDateTime(auth.user.last_login_at) }}
        </el-descriptions-item>
      </el-descriptions>
      <div class="language-row">
        <span>{{ $t('account.info.language') }}</span>
        <el-select
          :model-value="locale"
          class="language-select"
          :aria-label="$t('account.info.language')"
          @change="changeLocale"
        >
          <el-option
            value="zh-Hans"
            label="简体中文"
          />
          <el-option
            value="zh-Hant"
            label="繁體中文"
          />
          <el-option
            value="en"
            label="English"
          />
        </el-select>
      </div>
    </el-card>

    <el-card
      class="app-card app-card--danger"
      shadow="never"
    >
      <template #header>
        <div class="app-card__header">
          <div>
            <h2 class="app-card__title">
              {{ $t('account.erase.title') }}
            </h2>
            <p class="app-card__subtitle">
              {{ $t('account.erase.subtitle') }}
            </p>
          </div>
        </div>
      </template>
      <el-button
        type="danger"
        :loading="erasing"
        @click="eraseAccount"
      >
        {{ $t('account.erase.title') }}
      </el-button>
    </el-card>

    <h2 class="section-title">
      {{ $t('account.devices') }}
    </h2>

    <el-result
      v-if="error && !sessions.length"
      icon="warning"
      :title="$t('account.sessions.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="load"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>
    <el-alert
      v-else-if="error && sessions.length"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <el-skeleton
      v-else-if="loading && !sessions.length"
      :rows="4"
      animated
      aria-busy="true"
    />

    <el-empty
      v-else-if="!loading && !sessions.length"
      :description="$t('account.sessions.empty')"
    />

    <el-card
      v-else
      class="app-card"
      shadow="never"
    >
      <template #header>
        <div class="app-card__header">
          <h2 class="app-card__title">
            {{ $t('account.sessions.title') }}
          </h2>
          <div class="app-card__actions">
            <el-button
              size="small"
              :disabled="sessions.filter(s => !s.is_current).length === 0"
              :loading="revokingAll"
              @click="revokeAll"
            >
              {{ $t('account.sessions.revokeOthers') }}
            </el-button>
          </div>
        </div>
      </template>
      <ul class="session-list">
        <li
          v-for="item in sessions"
          :key="item.id"
          class="session-item"
        >
          <div class="session-info">
            <div class="session-device">
              {{ deviceLabel(item.user_agent) }}
              <el-tag
                v-if="item.is_current"
                type="success"
                size="small"
                effect="light"
              >
                {{ $t('account.sessions.current') }}
              </el-tag>
            </div>
            <div class="session-meta">
              {{ $t('account.sessions.meta', { lastSeen: formatDateTime(item.last_seen_at), created: formatDateTime(item.created_at) }) }}
            </div>
          </div>
          <el-button
            v-if="item.is_current"
            size="small"
            link
            type="danger"
            :loading="revokingId === item.id"
            @click="logout"
          >
            {{ $t('nav.logout') }}
          </el-button>
          <el-button
            v-else
            size="small"
            link
            type="danger"
            :loading="revokingId === item.id"
            @click="revoke(item.id)"
          >
            {{ $t('account.sessions.revoke') }}
          </el-button>
        </li>
      </ul>
    </el-card>

    <div class="legal-links">
      <a
        :href="`/${locale}/privacy/`"
        target="_blank"
        rel="noopener"
      >{{ $t('account.legal.privacy') }}</a>
      <a
        :href="`/${locale}/terms/`"
        target="_blank"
        rel="noopener"
      >{{ $t('account.legal.terms') }}</a>
      <a
        :href="`/${locale}/disclaimer/`"
        target="_blank"
        rel="noopener"
      >{{ $t('account.legal.disclaimer') }}</a>
    </div>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { fetchSessions, revokeSession, revokeAllSessions, type SessionInfo } from '@/api/me'
import { erasureAccount } from '@/api/account'
import { formatDateTime } from '@/utils/format'
import { setAppLocale, type AppLocale } from '@/i18n'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const auth = useAuthStore()
const router = useRouter()
const { locale, t } = useI18n()

const sessions = ref<SessionInfo[]>([])
const loading = ref(false)
const error = ref('')
const revoking = ref(false)
const revokingAll = ref(false)
const revokingId = ref('')
const erasing = ref(false)

function statusLabel(status?: string) {
  const labels: Record<string, string> = {
    active: t('account.status.active'),
    disabled: t('account.status.disabled'),
    deleted: t('account.status.deleted'),
  }
  return status ? labels[status] ?? status : '—'
}

function deviceLabel(userAgent: string) {
  if (!userAgent) return t('account.devicesUnknown')
  const ua = userAgent.toLowerCase()
  if (ua.includes('iphone') || ua.includes('android')) return t('account.device.mobile')
  if (ua.includes('ipad')) return t('account.device.tablet')
  return t('account.device.desktop')
}

async function changeLocale(value: AppLocale) {
  await setAppLocale(value)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    sessions.value = await fetchSessions()
  } catch (err) {
    error.value = toUserMessage(err, t('account.sessions.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function revoke(id: string) {
  try {
    await ElMessageBox.confirm(
      t('account.sessions.revokeConfirm'),
      t('account.sessions.revokeTitle'),
      { type: 'warning', confirmButtonText: t('account.sessions.revoke'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  revokingId.value = id
  try {
    await revokeSession(id)
    await load()
  } catch (err) {
    error.value = toUserMessage(err, t('account.sessions.revokeFailed'))
  } finally {
    revokingId.value = ''
  }
}

async function revokeAll() {
  try {
    await ElMessageBox.confirm(
      t('account.sessions.revokeAllConfirm'),
      t('account.sessions.revokeOthers'),
      { type: 'warning', confirmButtonText: t('account.sessions.revokeOthers'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  revokingAll.value = true
  try {
    await revokeAllSessions()
    await load()
  } catch (err) {
    error.value = toUserMessage(err, t('account.sessions.revokeFailed'))
  } finally {
    revokingAll.value = false
  }
}

async function logout() {
  revoking.value = true
  try {
    await auth.logout()
    await router.push({ name: 'login' })
  } finally {
    revoking.value = false
  }
}

async function eraseAccount() {
  try {
    await ElMessageBox.confirm(
      t('account.erase.confirm'),
      t('account.erase.title'),
      { type: 'error', confirmButtonText: t('account.erase.confirmAction'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  erasing.value = true
  try {
    await erasureAccount()
    await auth.logout()
    await router.push({ name: 'login' })
  } catch (err) {
    error.value = toUserMessage(err, t('account.erase.failed'))
  } finally {
    erasing.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--liki-font-serif);
  font-size: 17px;
  font-weight: 700;
  margin: 24px 0 12px;
}
.section-title::before {
  content: '';
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--liki-primary);
}
.sessions-head { display: flex; align-items: center; justify-content: space-between; }
.language-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 16px;
  color: var(--liki-text-2);
  font-size: 14px;
}
.language-select { width: 180px; }
.session-list { list-style: none; margin: 0; padding: 0; }
.session-item {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 12px 0; border-bottom: 1px solid var(--liki-line);
}
.session-item:last-child { border-bottom: none; }
.session-device { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.session-meta { color: var(--liki-text-2); font-size: 13px; margin-top: 2px; }
.legal-links { display: flex; gap: 18px; justify-content: center; margin-top: 28px; padding-top: 16px; border-top: 1px solid var(--liki-line); }
.legal-links a { color: var(--liki-text-2); font-size: 13px; text-decoration: none; padding: 6px 2px; }
.legal-links a:hover { color: var(--liki-primary); }
</style>
