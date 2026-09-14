<template>
  <el-card class="paipan-preview" shadow="never">
    <template #header>
      <div class="preview-head">
        <span class="card-title">免费排盘预览</span>
        <el-button size="small" type="primary" :loading="loading" @click="compute">排盘</el-button>
      </div>
    </template>

    <p v-if="profile.calendar === 'lunar'" class="hint">
      免费预览暂支持公历出生时间；农历可在付费对话中获得完整排盘与解读。
    </p>

    <template v-else>
      <AppAlert v-if="error" :title="error" type="error" />
      <p v-if="!result && !loading" class="hint">
        输入公历出生时间后点"排盘"，即可免费查看八字四柱（临时展示，不保存）。
      </p>

      <div v-if="loading" class="pillars">
        <div v-for="i in 4" :key="i" class="pillar"><div class="skel" /><div class="skel short" /></div>
      </div>

      <template v-if="result">
        <div class="pillars">
          <div v-for="item in pillars" :key="item.label" class="pillar">
            <div class="label">{{ item.label }}</div>
            <div class="gl font-serif">{{ item.gan }}{{ item.zhi }}</div>
            <div class="nayin">{{ item.naYin }}</div>
          </div>
        </div>
        <div v-if="correction" class="correction">{{ correction }}</div>
        <p v-if="result.zi_shi_rule" class="hint">换日口径：{{ result.zi_shi_rule }}</p>
      </template>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { engineBaziChart, engineCityCoords, engineSolarTime, type BaziChartData } from '@/api/engine'
import type { Profile } from '@/api/profiles'
import AppAlert from '@/components/app/AppAlert.vue'

const props = defineProps<{ profile: Profile }>()

const loading = ref(false)
const error = ref('')
const result = ref<BaziChartData | null>(null)
const correction = ref('')

const pillars = computed(() => {
  if (!result.value) return []
  const { nian, yue, ri, shi } = result.value
  return [
    { label: '年柱', gan: nian.gan, zhi: nian.zhi, naYin: nian.na_yin ?? '' },
    { label: '月柱', gan: yue.gan, zhi: yue.zhi, naYin: yue.na_yin ?? '' },
    { label: '日柱', gan: ri.gan, zhi: ri.zhi, naYin: ri.na_yin ?? '' },
    { label: '时柱', gan: shi.gan, zhi: shi.zhi, naYin: shi.na_yin ?? '' },
  ]
})

async function compute() {
  const p = props.profile
  error.value = ''
  correction.value = ''
  result.value = null
  if (p.calendar === 'lunar') return
  if (p.birth_time_known === 'unknown') {
    error.value = '出生时间不确定，暂无法预览排盘；可在付费对话中定盘。'
    return
  }
  if (!p.birth_time) {
    error.value = '缺少出生时间'
    return
  }
  loading.value = true
  try {
    const datetime = `${p.birth_date}T${p.birth_time}:00+08:00`
    let solarTime = datetime
    if (p.birth_time_known === 'known' && p.birth_place) {
      const coords = await engineCityCoords(p.birth_place)
      const t = await engineSolarTime(datetime, coords.longitude)
      solarTime = t.solar
      correction.value = '已做真太阳时修正'
    } else if (p.birth_time_known === 'known') {
      correction.value = '未做真太阳时修正（未填出生地）'
    } else {
      correction.value = '按已定时辰排盘，未做校正'
    }
    result.value = await engineBaziChart(solarTime, p.gender)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '排盘失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.paipan-preview { border-radius: 12px; margin-bottom: 20px; }
.preview-head { display: flex; align-items: center; justify-content: space-between; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); font-size: 13px; margin: 4px 0 12px; }
.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: 6px; overflow: hidden; margin-top: 12px;
}
.pillar { background: var(--liki-surface); padding: 14px 8px; text-align: center; }
.pillar .label { font-size: 12px; color: var(--liki-text-3); margin-bottom: 6px; }
.pillar .gl { font-size: 22px; font-weight: 700; }
.pillar .nayin { font-size: 12px; color: var(--liki-text-2); margin-top: 2px; }
.correction { margin-top: 10px; font-size: 12px; color: var(--liki-gold); }
.skel { height: 12px; border-radius: 6px; background: linear-gradient(90deg,#f0efeb 25%,#e7e4de 50%,#f0efeb 75%); background-size: 200% 100%; animation: shimmer 1.2s infinite; margin-bottom: 8px; }
.skel.short { width: 60%; margin: 0 auto; }
@keyframes shimmer { to { background-position: -200% 0; } }
@media (max-width: 640px) { .pillars { grid-template-columns: repeat(2, 1fr); } }
</style>