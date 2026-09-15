<template>
  <section>
    <AppPageHeader title="账户" subtitle="账号信息与登录设备管理。">
      <template #actions>
        <router-link to="/orders"><el-button>订单</el-button></router-link>
        <el-button :loading="revoking" type="danger" plain @click="logout">退出登录</el-button>
      </template>
    </AppPageHeader>

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">账号信息</span></template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="邮箱">{{ auth.user?.email ?? '—' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="auth.user?.status === 'active' ? 'success' : 'warning'" size="small">
            {{ statusLabel(auth.user?.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="auth.user?.last_login_at" label="最近登录">
          {{ formatTime(auth.user.last_login_at) }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <h2 class="section-title">删除账号</h2>
    <el-card class="block danger-zone" shadow="never">
      <p class="danger-hint">删除后，你的全部档案与对话中的个人信息将被匿名化清除（不可恢复）。</p>
      <el-button type="danger" :loading="erasing" @click="eraseAccount">删除账号（合规删除）</el-button>
    </el-card>

    <h2 class="section-title">登录设备</h2>

    <AppErrorState v-if="error && !sessions.length" title="会话加载失败" :description="error" @retry="load" />
    <AppAlert v-else-if="error && sessions.length" :title="error" type="error" />

    <AppSkeleton v-if="loading && !sessions.length" :rows="4" />

    <AppEmpty v-else-if="!loading && !sessions.length" description="没有登录会话" />

    <el-card v-else class="block" shadow="never">
      <template #header>
        <div class="sessions-head">
          <span class="card-title">登录会话</span>
          <el-button size="small" :disabled="sessions.filter(s => !s.is_current).length === 0" :loading="revokingAll" @click="revokeAll">
            撤销其他设备
          </el-button>
        </div>
      </template>
      <ul class="session-list">
        <li v-for="item in sessions" :key="item.id" class="session-item">
          <div class="session-info">
            <div class="session-device">
              {{ deviceLabel(item.user_agent) }}
              <el-tag v-if="item.is_current" type="success" size="small" effect="light">当前设备</el-tag>
            </div>
            <div class="session-meta">
              最近活跃：{{ formatTime(item.last_seen_at) }} · 登录于 {{ formatTime(item.created_at) }}
            </div>
          </div>
          <el-button
            v-if="item.is_current"
            size="small"
            text
            type="danger"
            :loading="revokingId === item.id"
            @click="logout"
          >
            退出
          </el-button>
          <el-button
            v-else
            size="small"
            text
            type="danger"
            :loading="revokingId === item.id"
            @click="revoke(item.id)"
          >
            撤销
          </el-button>
        </li>
      </ul>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { fetchSessions, revokeSession, revokeAllSessions, type SessionInfo } from '@/api/me'
import { erasureAccount } from '@/api/account'
import AppAlert from '@/components/app/AppAlert.vue'
import AppEmpty from '@/components/app/AppEmpty.vue'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const auth = useAuthStore()
const router = useRouter()

const sessions = ref<SessionInfo[]>([])
const loading = ref(false)
const error = ref('')
const revoking = ref(false)
const revokingAll = ref(false)
const revokingId = ref('')
const erasing = ref(false)

function statusLabel(status?: string) {
  const labels: Record<string, string> = { active: '正常', disabled: '已停用', deleted: '已删除' }
  return status ? labels[status] ?? status : '—'
}

function deviceLabel(userAgent: string) {
  if (!userAgent) return '未知设备'
  const ua = userAgent.toLowerCase()
  if (ua.includes('iphone') || ua.includes('android')) return '手机'
  if (ua.includes('ipad')) return '平板'
  return '电脑'
}

function formatTime(value: string) {
  if (!value) return '—'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    sessions.value = await fetchSessions()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载会话失败'
  } finally {
    loading.value = false
  }
}

async function revoke(id: string) {
  revokingId.value = id
  try {
    await revokeSession(id)
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '撤销失败'
  } finally {
    revokingId.value = ''
  }
}

async function revokeAll() {
  revokingAll.value = true
  try {
    await revokeAllSessions()
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '撤销失败'
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
      '确定删除账号？你的全部档案与对话中的个人信息将被匿名化清除，不可恢复。',
      '删除账号',
      { type: 'error', confirmButtonText: '删除', cancelButtonText: '取消' },
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
    error.value = err instanceof Error ? err.message : '删除失败'
  } finally {
    erasing.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.block { margin-bottom: 20px; border-radius: 12px; }
.card-title { font-weight: 600; }
.section-title { font-size: 18px; margin: 24px 0 12px; }
.sessions-head { display: flex; align-items: center; justify-content: space-between; }
.session-list { list-style: none; margin: 0; padding: 0; }
.session-item {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 12px 0; border-bottom: 1px solid var(--liki-line);
}
.session-item:last-child { border-bottom: none; }
.session-device { display: flex; align-items: center; gap: 8px; font-weight: 600; }
.session-meta { color: var(--liki-text-2); font-size: 13px; margin-top: 2px; }
.danger-zone { border-color: #f3c9c9; }
.danger-hint { color: var(--liki-text-2); font-size: 13px; margin: 0 0 14px; }
</style>