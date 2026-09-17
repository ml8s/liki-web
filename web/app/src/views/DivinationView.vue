<template>
  <section>
    <AppPageHeader title="问卦" subtitle="输入一个问题摇卦，先看卦象；深入断卦解读进对话。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">摇卦</span></template>
      <el-form label-position="top" @submit.prevent="cast">
        <el-form-item label="你想问的一件事">
          <el-input v-model="question" placeholder="如：这个项目能不能成 / 何时有结果" />
        </el-form-item>
        <el-button type="primary" :loading="loading" :disabled="!question.trim()" @click="cast">摇卦</el-button>
      </el-form>
    </el-card>

    <template v-if="snapshot">
      <el-card class="block" shadow="never">
        <template #header><span class="card-title">卦象</span></template>
        <div class="gua-row">
          <div class="gua-name font-serif">{{ guaName }}</div>
          <div class="gua-sub">{{ guaSub }}</div>
        </div>
        <div v-if="focusText" class="rel-sec">
          <div class="sec-title">用神</div>
          <div class="rel-text">{{ focusText }}</div>
        </div>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">深入解读</span></template>
        <p v-if="purchased" class="hint">进入对话，由命理顾问结合卦象为你深入断卦。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得深入断卦解读。</p>
        <p v-if="!profiles.list.length" class="hint">问卦解读需要一个档案，先<a href="/app/profiles/new">建立档案</a>。</p>
        <el-button v-if="purchased && profiles.list.length" type="primary" size="large" @click="startChat">深入解读</el-button>
        <el-button v-else type="primary" size="large" @click="router.push(profiles.list.length ? '/chat' : '/profiles/new')">
          {{ purchased ? (profiles.list.length ? '' : '建立档案') : '解锁 Chat' }}
        </el-button>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useProfilesStore } from '@/stores/profiles'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useChatStore } from '@/stores/chat'
import { engineLiuyaoSnapshot, type LiuyaoSnapshotResult } from '@/api/engine'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()

const question = ref('')
const snapshot = ref<LiuyaoSnapshotResult | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const guaName = computed(() => snapshot.value?.board?.name ?? '—')
const guaSub = computed(() => {
  const b = snapshot.value?.board
  const parts: string[] = []
  if (b?.palace) parts.push(`${b.palace}宫`)
  if (b?.xun_kong) parts.push(`旬空 ${b.xun_kong}`)
  return parts.join(' · ')
})
const focusText = computed(() => {
  const f = snapshot.value?.focus
  if (!f) return ''
  const parts: string[] = []
  if (f.yong_shen) parts.push(`用神 ${f.yong_shen}`)
  if (f.world) parts.push(`世 ${f.world}`)
  if (f.other) parts.push(`应 ${f.other}`)
  return parts.join(' · ')
})

async function cast() {
  if (!question.value.trim()) return
  loading.value = true
  error.value = ''
  snapshot.value = null
  try {
    snapshot.value = await engineLiuyaoSnapshot(question.value.trim(), 'auto')
  } catch (err) {
    error.value = toUserMessage(err, '摇卦失败')
  } finally {
    loading.value = false
  }
}

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  chat.pendingIntent = `我想问卦。问题：${question.value.trim()}`
  const existing = chat.sessions.find(s => s.profile_id === profile.id)
  if (existing) {
    void router.push(`/chat/${existing.id}`)
  } else {
    void chat.createSession(profile.id, 'liki').then(s => router.push(`/chat/${s.id}`)).catch(err => { error.value = toUserMessage(err, '创建会话失败') })
  }
}

onMounted(async () => {
  await Promise.all([profiles.loadList(), entitlements.load(), chat.loadSessions()])
})
</script>

<style scoped>
.block { border-radius: 12px; margin-bottom: 20px; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
.gua-row { display: flex; align-items: baseline; gap: 12px; }
.gua-name { font-size: 26px; font-weight: 700; }
.gua-sub { color: var(--liki-text-2); font-size: 13px; }
.rel-sec { margin-top: 14px; }
.sec-title { font-size: 13px; font-weight: 600; margin-bottom: 6px; }
.rel-text { font-size: 14px; color: var(--liki-ink); }
</style>