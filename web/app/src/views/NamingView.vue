<template>
  <section>
    <AppPageHeader title="起名" subtitle="依八字用神选字——先看喜用五行与候选字，定制化起名进对话。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">选择档案（起名对象）</span></template>
      <el-select v-model="profileId" aria-label="选择档案" placeholder="选择档案" style="width: 100%" @change="profileId && loadPreview()">
        <el-option v-for="p in profiles.list" :key="p.id" :value="p.id" :label="p.nickname" />
      </el-select>
      <p v-if="!profiles.list.length" class="hint">还没有档案，先<a href="/app/profiles/new">建立档案</a>再起名。</p>
    </el-card>

    <el-card v-if="loading" class="block" shadow="never"><template #header><span class="card-title">喜用五行</span></template>排盘中…</el-card>

    <template v-if="preview">
      <el-card class="block pan" shadow="never">
        <template #header><span class="card-title">喜用五行</span></template>
        <div class="ys">{{ preview.yongText }}</div>
        <div v-if="singlePool.length" class="sec-title">单字候选</div>
        <div v-if="singlePool.length" class="chars"><span v-for="c in singlePool" :key="c.char" class="char">{{ c.char }}</span></div>
        <div v-if="doublePool.length" class="sec-title">双字候选</div>
        <div v-if="doublePool.length" class="chars"><span v-for="c in doublePool" :key="c.char" class="char">{{ c.char }}</span></div>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">定制化起名</span></template>
        <p v-if="purchased" class="hint">进入对话，命理顾问结合八字为你定制名字。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得定制化起名。</p>
        <el-button v-if="purchased" type="primary" size="large" @click="startChat">定制化起名</el-button>
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
import { engineBaziFullChart, engineQimingPick, type BaziYongShen } from '@/api/engine'
import { baziChartFor } from '@/utils/paipan'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()

const profileId = ref('')
const preview = ref<{ yongText: string; singlePool: Array<{ char: string }>; doublePool: Array<{ char: string }> } | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

const singlePool = computed(() => preview.value?.singlePool ?? [])
const doublePool = computed(() => preview.value?.doublePool ?? [])

function yongXi(ys?: BaziYongShen): { yong: string; xi: string } {
  const f = ys?.fu_yi
  if (f?.yong) return { yong: f.yong, xi: f.xi ?? f.yong }
  const g = ys?.ge_ju
  if (g?.yong) return { yong: g.yong, xi: g.xi ?? g.yong }
  const t = ys?.tiao_hou
  if (t?.yong) return { yong: t.yong, xi: t.xi ?? t.yong }
  return { yong: '', xi: '' }
}

async function loadPreview() {
  const p = profile.value
  if (!p) return
  loading.value = true
  error.value = ''
  preview.value = null
  try {
    const chart = await baziChartFor(p)
    const full = await engineBaziFullChart(chart)
    const { yong, xi } = yongXi(full.yong_shen)
    if (!yong) throw new Error('未评估出喜用神，暂无法取字')
    const qiangruo = full.yong_shen?.fu_yi?.qiangruo ?? ''
    const yongText = `${qiangruo}${qiangruo ? ' · ' : ''}喜用 ${yong}${xi && xi !== yong ? `/${xi}` : ''}`
    const [single, double] = await Promise.all([
      engineQimingPick(yong, xi !== yong ? xi : '', 1),
      engineQimingPick(yong, xi !== yong ? xi : '', 2),
    ])
    const poolChars = (r: typeof single) => (r.pools?.[0]?.chars ?? [])
    preview.value = { yongText, singlePool: poolChars(single), doublePool: poolChars(double) }
  } catch (err) {
    error.value = toUserMessage(err, '起名预览失败')
  } finally {
    loading.value = false
  }
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = '我想咨询起名。'
  void chat.ensureSession(p.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, '创建会话失败') })
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
.ys { font-size: 15px; color: var(--liki-ink); }
.sec-title { font-size: 13px; font-weight: 600; margin: 14px 0 8px; }
.chars { display: flex; flex-wrap: wrap; gap: 8px; }
.char { display: grid; place-items: center; width: 44px; height: 44px; border: 1px solid var(--liki-line); border-radius: 8px; font-size: 20px; font-family: var(--liki-font-serif, serif); background: var(--liki-surface); }
</style>