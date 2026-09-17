<template>
  <section>
    <AppPageHeader title="合盘" subtitle="用你的档案与对方信息，查看两人的八字合盘。" />

    <AppAlert v-if="error" :title="error" type="error" />

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">第一步 · 选择档案（你或对方）</span></template>
      <el-select v-model="profileAId" aria-label="选择档案" placeholder="选择档案" style="width: 100%" @change="profileAId && loadA()">
        <el-option v-for="p in profiles.list" :key="p.id" :value="p.id" :label="p.nickname" />
      </el-select>
      <p v-if="!profiles.list.length" class="hint">还没有档案，先<a href="/app/profiles/new">建立档案</a>再合盘。</p>
    </el-card>

    <el-card class="block" shadow="never">
      <template #header><span class="card-title">第二步 · 输入对方信息</span></template>
      <el-form label-position="top" class="b-form">
        <div class="row">
          <el-form-item label="性别">
            <el-radio-group v-model="b.gender">
              <el-radio-button value="female">女</el-radio-button>
              <el-radio-button value="male">男</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="出生日期">
            <el-date-picker v-model="b.birth_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" style="width: 100%" />
          </el-form-item>
          <el-form-item label="出生时间">
            <el-time-picker v-model="b.birth_time" format="HH:mm" value-format="HH:mm" placeholder="如 06:00" style="width: 100%" />
          </el-form-item>
          <el-form-item label="出生地（可选）">
            <el-input v-model="b.birth_place" placeholder="用于真太阳时修正，如：上海" />
          </el-form-item>
        </div>
      </el-form>
      <el-button type="primary" :loading="loading" :disabled="!profileAId" @click="compute">合盘</el-button>
    </el-card>

    <template v-if="result">
      <el-card class="block" shadow="never">
        <template #header><span class="card-title">合盘结果</span></template>

        <div class="charts">
          <div class="chart-row">
            <div class="person">{{ aName }}</div>
            <div class="pillars">
              <div v-for="p in pillarsA" :key="p.label" class="pillar">
                <div class="pl">{{ p.label }}</div>
                <div class="gl font-serif">{{ p.gan }}{{ p.zhi }}</div>
                <div class="sub">{{ p.naYin }}</div>
              </div>
            </div>
          </div>
          <div class="chart-row">
            <div class="person">对方</div>
            <div class="pillars">
              <div v-for="p in pillarsB" :key="p.label" class="pillar">
                <div class="pl">{{ p.label }}</div>
                <div class="gl font-serif">{{ p.gan }}{{ p.zhi }}</div>
                <div class="sub">{{ p.naYin }}</div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="dayMasterRel" class="rel-sec">
          <div class="sec-title">日主关系</div>
          <div class="rel-text">{{ dayMasterRel }}</div>
        </div>

        <div v-if="keyRels.length" class="rel-sec">
          <div class="sec-title">关键地支关系</div>
          <div class="rels">
            <span v-for="r in keyRels" :key="r" class="rel-chip">{{ r }}</span>
          </div>
        </div>

        <div v-if="shiShenRows.length" class="rel-sec">
          <div class="sec-title">十神互看</div>
          <div class="rels">
            <span v-for="r in shiShenRows" :key="r" class="rel-chip">{{ r }}</span>
          </div>
        </div>
      </el-card>

      <el-card class="block" shadow="never">
        <template #header><span class="card-title">合盘解读</span></template>
        <p v-if="purchased" class="hint">开启对话，由命理顾问为你解读两人的合盘。</p>
        <p v-else class="hint">解锁 Liki Chat 后可获得专业合盘解读。</p>
        <el-button v-if="purchased" type="primary" size="large" @click="startInterpretation">开始合盘解读</el-button>
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
import {
  engineBaziBond,
  engineBaziChart,
  engineCityCoords,
  engineSolarTime,
  type BaziBondResult,
  type BaziChartData,
} from '@/api/engine'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()
const chat = useChatStore()

const profileAId = ref('')
const chartA = ref<BaziChartData | null>(null)
const bChart = ref<BaziChartData | null>(null)
const b = ref({ gender: 'female', birth_date: '', birth_time: '', birth_place: '' })
const result = ref<BaziBondResult | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const profileA = computed(() => profiles.list.find(p => p.id === profileAId.value) ?? null)
const aName = computed(() => profileA.value?.nickname ?? '档案 A')

const PILLAR_LABELS = ['年柱', '月柱', '日柱', '时柱']
function pillarsOf(chart: BaziChartData | null) {
  if (!chart) return []
  const list = [chart.nian, chart.yue, chart.ri, chart.shi]
  return PILLAR_LABELS.map((label, i) => ({
    label,
    gan: list[i]?.gan ?? '',
    zhi: list[i]?.zhi ?? '',
    naYin: list[i]?.na_yin ?? '',
  }))
}
const pillarsA = computed(() => pillarsOf(chartA.value))
const pillarsB = computed(() => pillarsOf(bChart.value))

async function loadA() {
  error.value = ''
  result.value = null
  try {
    const p = profileA.value
    if (!p) return
    if (p.calendar === 'lunar' || p.birth_time_known === 'unknown' || !p.birth_time) {
      throw new Error('档案需公历出生时间才能合盘')
    }
    chartA.value = await chartFor(p.gender, p.birth_date, p.birth_time, p.birth_time_known, p.birth_place)
  } catch (err) {
    error.value = toUserMessage(err, '读取档案失败')
  }
}

async function chartFor(gender: string, birthDate: string, birthTime: string, known: string, place: string) {
  let longitude = 120
  if (known === 'known' && place) {
    const coords = await engineCityCoords(place)
    longitude = coords.longitude
  }
  const datetime = `${birthDate}T${birthTime}:00+08:00`
  const t = await engineSolarTime(datetime, longitude)
  const solar = known === 'known' && place ? t.solar : datetime
  return engineBaziChart(solar, gender)
}

async function compute() {
  error.value = ''
  result.value = null
  bChart.value = null
  if (!profileAId.value) return
  if (!b.value.birth_date || !b.value.birth_time) {
    error.value = '请填写对方的出生日期与时间'
    return
  }
  loading.value = true
  try {
    if (!chartA.value) await loadA()
    if (!chartA.value) return
    bChart.value = await chartFor(b.value.gender, b.value.birth_date, b.value.birth_time, 'known', b.value.birth_place)
    result.value = await engineBaziBond(chartA.value, bChart.value)
  } catch (err) {
    error.value = toUserMessage(err, '合盘失败')
  } finally {
    loading.value = false
  }
}

const ZHU_LABELS: Record<string, string> = { nian: '年柱', yue: '月柱', ri: '日柱', shi: '时柱' }

function relText(rel: { relation?: string; detail?: string; type?: string }): string {
  return rel.relation || rel.detail || rel.type || ''
}

const dayMasterRel = computed(() => {
  const pairs = result.value?.zhu_cross?.pairs ?? []
  const pair = pairs.find(p => p.jia_zhu === 'ri' && p.yi_zhu === 'ri')
  if (!pair) return ''
  return `${aName.value} 日主 ${pair.jia_gan}${pair.jia_zhi} ↔ 对方日主 ${pair.yi_gan}${pair.yi_zhi} · 天干 ${relText(pair.gan_guan_xi)} · 地支 ${relText(pair.zhi_guan_xi)}`
})

const keyRels = computed(() => {
  const pairs = result.value?.zhu_cross?.pairs ?? []
  const set = new Set<string>()
  for (const p of pairs) {
    const t = relText(p.zhi_guan_xi)
    if (/六合|三合|六冲|六害|相刑/.test(t)) {
      set.add(`${ZHU_LABELS[p.jia_zhu] ?? p.jia_zhu}·${p.jia_gan}${p.jia_zhi} — ${ZHU_LABELS[p.yi_zhu] ?? p.yi_zhu}·${p.yi_gan}${p.yi_zhi}：${t}`)
    }
  }
  return [...set].slice(0, 8)
})

const shiShenRows = computed(() => {
  const cross = result.value?.shi_shen_cross
  if (!cross) return []
  const rows: string[] = []
  const short: Record<string, string> = { nian: '年', yue: '月', ri: '日', shi: '时' }
  for (const k of ['nian_gan', 'yue_gan', 'ri_gan', 'shi_gan']) {
    const a = cross.jia_dui_yi?.[k]
    const bb = cross.yi_dui_jia?.[k]
    const label = short[k.replace('_gan', '')] ?? k.replace('_gan', '')
    if (a) rows.push(`${aName.value}${label}柱看对方：${a}`)
    if (bb) rows.push(`对方${label}柱看${aName.value}：${bb}`)
  }
  return rows
})

async function startInterpretation() {
  const p = profileA.value
  const bProfile = b.value
  if (!p) return
  const bInfo = `${bProfile.gender === 'male' ? '男' : '女'}，公历 ${bProfile.birth_date} ${bProfile.birth_time}${bProfile.birth_place ? `，出生地 ${bProfile.birth_place}` : ''}`
  chat.pendingIntent = `请为我（${p.nickname}）与对方做合盘解读。对方信息：${bInfo}。`
  const existing = chat.sessions.find(s => s.profile_id === p.id)
  if (existing) {
    void router.push(`/chat/${existing.id}`)
  } else {
    try {
      const session = await chat.createSession(p.id, 'liki')
      void router.push(`/chat/${session.id}`)
    } catch (err) {
      error.value = toUserMessage(err, '创建会话失败')
    }
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
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; }
.charts { display: grid; gap: 14px; }
.chart-row { display: flex; align-items: center; gap: 12px; }
.person { flex: 0 0 56px; font-weight: 600; font-size: 14px; }
.pillars { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; flex: 1; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; }
.pillar { background: var(--liki-surface); padding: 8px 4px; text-align: center; }
.pillar .pl { font-size: 11px; color: var(--liki-text-3); }
.pillar .gl { font-size: 18px; font-weight: 700; }
.pillar .sub { font-size: 11px; color: var(--liki-text-2); }
.rel-sec { margin-top: 16px; }
.sec-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: var(--liki-text-1); }
.rel-text { font-size: 14px; color: var(--liki-ink); }
.rels { display: flex; flex-wrap: wrap; gap: 8px; }
.rel-chip { padding: 4px 10px; border-radius: 999px; border: 1px solid var(--liki-line); background: var(--liki-surface); font-size: 12px; color: var(--liki-text-2); }
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(4, 1fr); }
}
</style>