<template>
  <section>
    <AppPageHeader title="风水" subtitle="先看命卦与八宅吉凶方位、流年飞星；布局解读进对话。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">选择档案（本命）</span></template>
      <el-select v-model="profileId" aria-label="选择档案" placeholder="选择档案" style="width: 100%" @change="profileId && loadPreview()">
        <el-option v-for="p in profiles.list" :key="p.id" :value="p.id" :label="p.nickname" />
      </el-select>
      <p v-if="!profiles.list.length" class="hint">还没有档案，先<a href="/app/profiles/new">建立档案</a>再看风水。</p>
    </el-card>

    <el-card v-if="loading" class="block" shadow="never"><template #header><span class="card-title">命卦</span></template>排盘中…</el-card>

    <template v-if="bazhai">
      <el-card class="block" shadow="never">
        <template #header><span class="card-title">命卦与吉凶方</span></template>
        <div class="mingua-row">
          <div class="mingua font-serif">{{ bazhai.ming_gua.gua.name }}</div>
          <div class="mg-name">{{ bazhai.ming_gua.group }}</div>
        </div>
        <div class="dirs">
          <div v-for="d in bazhaiDirs" :key="d.name" :class="['dir', { good: d.good }]">
            <div class="dg font-serif">{{ d.gua }}</div>
            <div class="dname">{{ d.name }}</div>
          </div>
        </div>
        <template v-if="liuNianXing">
          <div class="sec-title">流年飞星（{{ liuNianXing.year }}）</div>
          <div class="ln-row">入中星：<strong>{{ liuNianXing.ru_zhong }}</strong></div>
          <div class="fly-grid">
            <div v-for="f in flyStars" :key="f.gong_num" :class="['fly-cell', { center: f.gong_num === 5 }]">
              <div class="fly-name">{{ f.xing_name }}</div>
            </div>
          </div>
        </template>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">布局解读</span></template>
        <p v-if="purchased" class="hint">进入对话，命理顾问结合命卦与飞星为你做布局解读。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得专业布局解读。</p>
        <el-button v-if="purchased" type="primary" size="large" @click="startChat">布局解读</el-button>
        <el-button v-else type="primary" size="large" @click="router.push('/chat')">解锁 Chat</el-button>
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
import { engineBazhaiChart, type BazhaiChart } from '@/api/engine'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()

const profileId = ref('')
const bazhai = ref<BazhaiChart | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

const BAZHAI_AUS = ['sheng_qi', 'tian_yi', 'yan_nian', 'fu_wei']
const bazhaiDirs = computed(() => {
  if (!bazhai.value) return []
  return Object.entries(bazhai.value.ba_zhai_dirs).map(([key, values]) => ({
    name: key, gua: (values ?? [''])[0], good: BAZHAI_AUS.includes(key),
  }))
})
const liuNianXing = computed(() => bazhai.value?.liu_nian_xing ?? null)
const flyStars = computed(() => {
  const gong = liuNianXing.value?.gong_wei
  if (!gong || gong.length !== 9) return []
  const order = [4, 9, 2, 3, 5, 7, 8, 1, 6]
  return order.map(n => gong.find(g => g.gong_num === n) ?? { gong_num: n, xing_name: '—' })
})

async function loadPreview() {
  const p = profile.value
  if (!p) return
  loading.value = true
  error.value = ''
  bazhai.value = null
  try {
    if (p.calendar === 'lunar') throw new Error('免费预览暂支持公历出生时间')
    const birthYear = parseInt(p.birth_date.slice(0, 4), 10)
    bazhai.value = await engineBazhaiChart(birthYear, p.gender)
  } catch (err) {
    error.value = toUserMessage(err, '风水预览失败')
  } finally {
    loading.value = false
  }
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = '我想咨询风水。'
  void chat.ensureSession(p.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, '创建会话失败') })
}

onMounted(async () => {
  await Promise.all([profiles.loadList(), entitlements.load(), chat.loadSessions()])
})
</script>

<style scoped>
.block { border-radius: 12px; margin-bottom: 20px; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
.mingua-row { display: flex; align-items: center; gap: 14px; margin-bottom: 14px; }
.mingua { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 50%; border: 2px solid var(--liki-gold); font-size: 24px; color: #8a6a2f; background: var(--liki-gold-tint); }
.mg-name { font-size: 15px; font-weight: 600; }
.dirs { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.dir { border: 1px solid var(--liki-line); border-radius: 6px; padding: 10px; text-align: center; }
.dir .dg { font-size: 16px; color: var(--liki-gold); }
.dir.good { border-color: #bfe6bf; background: #f4faf4; }
.dir.good .dg { color: var(--el-color-success); }
.dir .dname { font-size: 12px; color: var(--liki-text-2); margin-top: 3px; }
.sec-title { font-size: 13px; font-weight: 600; margin: 16px 0 8px; }
.ln-row { font-size: 13px; color: var(--liki-text-2); margin-bottom: 10px; }
.fly-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; width: min(240px, 100%); }
.fly-cell { background: var(--liki-surface); padding: 8px 4px; text-align: center; }
.fly-cell.center { background: var(--liki-gold-tint); }
.fly-name { font-size: 12px; color: var(--liki-text-2); }
@media (max-width: 640px) { .dirs { grid-template-columns: repeat(2, 1fr); } }
</style>