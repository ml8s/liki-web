<template>
  <section>
    <AppPageHeader title="择日" subtitle="选事项与日期范围，先看黄历推荐；择日解读进对话。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">择日</span></template>
      <el-form label-position="top" @submit.prevent="calc">
        <div class="row">
          <el-form-item label="事项">
            <el-select v-model="event" aria-label="择日事项" style="width: 100%">
              <el-option v-for="(label, key) in EVENTS" :key="key" :value="key" :label="label" />
            </el-select>
          </el-form-item>
          <el-form-item label="起始日期">
            <el-date-picker v-model="startDate" type="date" value-format="YYYY-MM-DD" placeholder="起始" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束日期">
            <el-date-picker v-model="endDate" type="date" value-format="YYYY-MM-DD" placeholder="结束" style="width: 100%" />
          </el-form-item>
        </div>
        <el-button type="primary" :loading="loading" :disabled="!event || !startDate || !endDate" @click="calc">择日</el-button>
      </el-form>
    </el-card>

    <template v-if="result">
      <el-card class="block pan" shadow="never">
        <template #header><span class="card-title">黄历推荐（{{ result.event_label || EVENTS[result.event || event] }}）</span></template>
        <div v-if="recommended.length" class="days">
          <div v-for="d in recommended" :key="d.date" class="day good">
            <div class="d-date">{{ d.date }}</div>
            <div class="d-sub">{{ d.jian_chu ? `${d.jian_chu} · ` : '' }}{{ d.reason || '' }}</div>
          </div>
        </div>
        <p v-if="!recommended.length" class="hint">该范围内没有推荐吉日。</p>
        <div v-if="unsuitable.length" class="sec-title">不宜</div>
        <div v-if="unsuitable.length" class="days">
          <div v-for="d in unsuitable" :key="d.date" class="day bad">
            <div class="d-date">{{ d.date }}</div>
            <div class="d-sub">{{ d.reason || '不宜' }}</div>
          </div>
        </div>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">择日解读</span></template>
        <p v-if="purchased" class="hint">进入对话，命理顾问结合黄历为你做择日解读。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得专业择日解读。</p>
        <p v-if="!profiles.list.length" class="hint">择日解读需要一个档案，先<a href="/app/profiles/new">建立档案</a>。</p>
        <el-button v-if="purchased && profiles.list.length" type="primary" size="large" @click="startChat">择日解读</el-button>
        <el-button v-else type="primary" size="large" @click="router.push(profiles.list.length ? '/chat' : '/profiles/new')">{{ purchased ? '建立档案' : '解锁 Chat' }}</el-button>
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
import { engineHuangliDays, type HuangliDaysResult } from '@/api/engine'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const EVENTS: Record<string, string> = {
  wedding: '结婚', engage: '订婚', opening: '开业', sign: '签约', move: '搬家',
  travel: '出行', build: '动土', exam: '考试', medical: '就医', sacrifice: '祭祀',
  cleaning: '扫除', renovation: '装修', bed_install: '安床', income: '纳财', funeral: '丧葬',
}

const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()

const event = ref('wedding')
const startDate = ref('')
const endDate = ref('')
const result = ref<HuangliDaysResult | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const recommended = computed(() => result.value?.recommended ?? [])
const unsuitable = computed(() => result.value?.unsuitable ?? [])

async function calc() {
  if (!event.value || !startDate.value || !endDate.value) return
  if (startDate.value > endDate.value) {
    error.value = '结束日期不能早于起始日期'
    return
  }
  loading.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await engineHuangliDays(EVENTS[event.value], event.value, startDate.value, endDate.value)
  } catch (err) {
    error.value = toUserMessage(err, '择日失败')
  } finally {
    loading.value = false
  }
}

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  chat.pendingIntent = `我想咨询择日（${EVENTS[event.value]}）。`
  void chat.ensureSession(profile.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, '创建会话失败') })
}

onMounted(async () => {
  await Promise.all([profiles.loadList(), entitlements.load(), chat.loadSessions()])
})
</script>

<style scoped>
.block { border-radius: 12px; margin-bottom: 20px; }
.block.pan { border-top: 2px solid var(--liki-gold); }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
.sec-title { font-size: 13px; font-weight: 600; margin: 16px 0 8px; }
.days { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 8px; }
.day { border: 1px solid var(--liki-line); border-radius: 8px; padding: 10px; text-align: center; }
.day.good { border-color: #bfe6bf; background: #f4faf4; }
.day.bad { border-color: #f3c9c9; background: #fdf7f7; }
.d-date { font-weight: 700; }
.d-sub { font-size: 12px; color: var(--liki-text-2); margin-top: 4px; }
</style>