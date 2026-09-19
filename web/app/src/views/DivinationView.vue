<template>
  <section class="service-view">
    <AppPageHeader title="问卦" subtitle="输入一个问题摇卦，先看卦象；深入断卦解读进对话。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">摇卦</span></template>
      <el-form label-position="top" @submit.prevent="cast">
        <el-form-item label="你想问的一件事">
          <el-input v-model="question" maxlength="200" placeholder="如：这个项目能不能成 / 何时有结果" show-word-limit />
        </el-form-item>
        <el-button type="primary" :loading="loading" :disabled="!question.trim()" @click="cast">摇卦</el-button>
      </el-form>
    </el-card>

    <template v-if="snapshot">
      <el-card class="block pan" shadow="never">
        <template #header><span class="card-title">卦象</span></template>
        <div class="gua-row">
          <div class="gua-name font-serif">{{ board.name ?? '—' }}</div>
          <div class="gua-sub">
            <span v-if="board.ben_gua">本卦 {{ board.ben_gua }}</span>
            <span v-if="board.bian_gua"> · 变卦 {{ board.bian_gua }}</span>
          </div>
        </div>
        <div class="gua-info">
          <span v-if="board.gong">{{ board.gong }}宫</span>
          <span v-if="board.gong_wuxing"> · {{ board.gong_wuxing }}</span>
          <span v-if="board.ri_chen_gan"> · 日辰 {{ board.ri_chen_gan }}{{ board.ri_chen_zhi }}</span>
          <span v-if="board.yue_jian_zhi"> · 月建 {{ board.yue_jian_gan }}{{ board.yue_jian_zhi }}</span>
          <span v-if="xunKongText"> · 旬空 {{ xunKongText }}</span>
        </div>

        <div v-if="lines.length" class="lines-sec">
          <div class="sec-title">六爻</div>
          <div class="lines-table-wrap">
            <div class="lines-table">
              <div class="line-row line-header">
                <span class="lc">爻位</span>
                <span class="lc">六亲</span>
                <span class="lc">地支</span>
                <span class="lc">五行</span>
                <span class="lc">天干</span>
                <span class="lc">旺衰</span>
                <span class="lc">日辰</span>
              </div>
              <div v-for="(line, idx) in lines" :key="idx" :class="['line-row', { dong: isDong(idx) }]">
                <span class="lc">{{ 6 - idx }}爻</span>
                <span class="lc">{{ line.liu_qin ?? '—' }}</span>
                <span class="lc">{{ line.di_zhi ?? '—' }}</span>
                <span class="lc">{{ line.xing ?? '—' }}</span>
                <span class="lc">{{ line.tian_gan ?? '—' }}</span>
                <span class="lc">{{ line.wang_shuai ?? '—' }}</span>
                <span class="lc">{{ line.ri_chen_relation ?? '—' }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="bianLines.length" class="lines-sec">
          <div class="sec-title">变爻</div>
          <div class="lines-table">
            <div v-for="(line, idx) in bianLines" :key="idx" class="line-row">
              <span class="lc">{{ 6 - idx }}爻</span>
              <span class="lc">{{ line.liu_qin ?? '—' }}</span>
              <span class="lc">{{ line.di_zhi ?? '—' }}</span>
              <span class="lc">{{ line.tian_gan ?? '—' }}</span>
            </div>
          </div>
        </div>

        <div v-if="focusText" class="rel-sec">
          <div class="sec-title">用神</div>
          <div class="rel-text">{{ focusText }}</div>
          <div v-if="yongShenDetail" class="rel-sub">{{ yongShenDetail }}</div>
        </div>

        <div v-if="guaCi" class="rel-sec">
          <div class="sec-title">卦辞</div>
          <div class="rel-text">{{ guaCi }}</div>
        </div>

        <div v-if="patterns.length" class="rel-sec">
          <div class="sec-title">格局</div>
          <div v-for="p in patterns" :key="p.name" class="pattern-item">
            <span class="pattern-name">{{ p.name }}</span>
            <span class="pattern-desc">{{ p.description }}</span>
          </div>
        </div>

        <div v-if="timingText" class="rel-sec">
          <div class="sec-title">应期</div>
          <div class="rel-text">{{ timingText }}</div>
        </div>

        <div v-if="snapshot.safety_advisory" class="safety-box">
          <AppAlert :title="snapshot.safety_advisory" type="warning" />
        </div>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">深入解读</span></template>
        <p v-if="entitlements.purchased" class="hint">进入对话，由命理顾问结合卦象为你深入断卦。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得深入断卦解读。</p>
        <p v-if="!profiles.list.length" class="hint">问卦解读需要一个档案，先<a href="/app/profiles/new">建立档案</a>。</p>
        <el-button v-if="entitlements.purchased && profiles.list.length" type="primary" size="large" @click="startChat">深入解读</el-button>
        <el-button v-else type="primary" size="large" @click="router.push(profiles.list.length ? '/chat' : '/profiles/new')">
          {{ entitlements.purchased ? (profiles.list.length ? '' : '建立档案') : '解锁 Chat' }}
        </el-button>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useServiceView } from '@/composables/useServiceView'
import { engineLiuyaoQigua, engineLiuyaoChart, engineSolarTime, type LiuyaoChartResult } from '@/api/engine'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const { profiles, entitlements, chat } = useServiceView()

const question = ref('')
const snapshot = ref<LiuyaoChartResult | null>(null)
const loading = ref(false)
const error = ref('')

const board = computed(() => snapshot.value?.board ?? {})
const lines = computed(() => snapshot.value?.board?.lines ?? [])
const bianLines = computed(() => snapshot.value?.board?.bian_yao ?? [])
const dongYao = computed(() => snapshot.value?.casting?.dong_yao ?? [])
const patterns = computed(() => snapshot.value?.patterns ?? [])

function isDong(idx: number) {
  const pos = 6 - idx
  return dongYao.value.includes(pos)
}

const xunKongText = computed(() => {
  const xk = snapshot.value?.board?.xun_kong
  if (!xk || !xk.length) return ''
  return xk.join(' ')
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

const yongShenDetail = computed(() => {
  const ys = snapshot.value?.yong_shen
  if (!ys) return ''
  const parts: string[] = []
  if (ys.name) parts.push(ys.name)
  if (ys.position) parts.push(`第${ys.position}爻`)
  return parts.length ? `用神为${parts.join('，')}` : ''
})

const guaCi = computed(() => {
  const gc = snapshot.value?.gua_ci
  if (!gc) return ''
  return [gc.gua, gc.ci].filter(Boolean).join('：')
})

const timingText = computed(() => {
  const tc = snapshot.value?.timing_candidates
  if (!tc || !tc.length) return ''
  return tc.map(t => [t.time, t.description].filter(Boolean).join('：')).join('；')
})

async function cast() {
  if (!question.value.trim()) return
  loading.value = true
  error.value = ''
  snapshot.value = null
  try {
    const { casting } = await engineLiuyaoQigua('auto')
    const now = new Date()
    const pad = (n: number) => String(n).padStart(2, '0')
    const solarTime = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}T${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}+08:00`
    const t = await engineSolarTime(solarTime, 120)
    snapshot.value = await engineLiuyaoChart(t.solar, casting)
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
  void chat.ensureSession(profile.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, '创建会话失败') })
}
</script>

<style scoped>
.gua-row { display: flex; align-items: baseline; gap: 12px; }
.gua-name { font-size: 26px; font-weight: 700; }
.gua-sub { color: var(--liki-text-2); font-size: 13px; }
.gua-info { margin-top: 8px; font-size: 13px; color: var(--liki-text-2); }
.lines-sec { margin-top: 14px; }
.lines-table-wrap { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.lines-table { border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; font-size: 12px; min-width: 480px; }
.line-row { display: flex; }
.line-row + .line-row { border-top: 1px solid var(--liki-line); }
.line-header { background: var(--liki-surface); font-weight: 600; }
.line-row .lc { flex: 1; padding: 6px 4px; text-align: center; }
.line-row.dong { background: var(--liki-gold-tint); }
.rel-sec { margin-top: 14px; }
.rel-text { font-size: 14px; color: var(--liki-ink); }
.rel-sub { font-size: 13px; color: var(--liki-text-2); margin-top: 4px; }
.pattern-item { display: flex; flex-direction: column; gap: 2px; margin-bottom: 4px; }
.pattern-name { font-weight: 600; color: var(--liki-gold); }
.pattern-desc { font-size: 12px; color: var(--liki-text-3); }
.safety-box { margin-top: 14px; }
</style>
