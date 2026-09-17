<template>
  <el-card class="paipan-preview" shadow="never">
    <template #header>
      <div class="preview-head">
        <span class="card-title">免费排盘</span>
        <el-button size="small" type="primary" :loading="loading" @click="compute">排盘</el-button>
      </div>
    </template>

    <p v-if="profile.calendar === 'lunar'" class="hint">
      免费预览暂支持公历出生时间；农历可在付费对话中获得完整排盘与解读。
    </p>

    <template v-else>
      <AppAlert v-if="error" :title="error" type="error" />
      <p v-if="!anyResult && !loading" class="hint">
        输入公历出生时间后点"排盘"，免费查看八字、紫微、八宅三盘（临时展示，不保存）。
      </p>

      <div v-if="loading" class="pan-skeleton">
        <div v-for="i in 6" :key="i" class="skel" />
      </div>

      <template v-if="anyResult">
        <el-tabs v-model="tab">
          <el-tab-pane label="八字" name="bazi">
            <div class="pillars">
              <div v-for="item in baziPillars" :key="item.label" :class="['pillar', { self: item.self }]">
                <div class="label">{{ item.label }}</div>
                <div class="gl font-serif">{{ item.gan }}{{ item.zhi }}</div>
                <div class="sub">{{ item.shiShen }}</div>
                <div class="sub nayin">{{ item.naYin }}</div>
                <div v-if="item.cangGan" class="sub cang">{{ item.cangGan }}</div>
                <span v-if="item.isVoid" class="void-tag">旬空</span>
              </div>
            </div>
            <div v-if="sanYuanText" class="san-yuan">
              <span class="sy-label">三元</span>
              <span class="sy-body">{{ sanYuanText }}</span>
            </div>
            <div v-if="shenShaList.length" class="shensha">
              <span class="sy-label">神煞</span>
              <span v-for="s in shenShaList" :key="s" class="ss-chip">{{ s }}</span>
            </div>
            <template v-if="daYunSteps.length">
              <div class="sec-title">大运</div>
              <div class="dayun">
                <div
                  v-for="(s, i) in daYunSteps"
                  :key="`${s.name}-${i}`"
                  :class="['dy', { cur: i === currentStepIndex }]"
                >
                  <div class="dy-name font-serif">{{ s.name }}</div>
                  <div class="dy-shen">{{ s.shi_shen }}</div>
                  <div class="dy-year">{{ s.start_year }}–{{ s.end_year }}</div>
                </div>
              </div>
            </template>
            <template v-if="yongShenSummary.length">
              <div class="sec-title">用神简览</div>
              <div class="ys">
                <div v-for="row in yongShenSummary" :key="row.label" class="ys-row">
                  <span class="ys-label">{{ row.label }}</span>
                  <span class="ys-body">{{ row.text }}</span>
                </div>
              </div>
            </template>
          </el-tab-pane>

          <el-tab-pane label="紫微" name="ziwei">
            <div class="palaces">
              <div v-for="g in ziweiGongs" :key="g.name" :class="['palace', { ming: g.name === '命宫', shen: g.is_shen_gong }]">
                <div class="pname">{{ g.name }}</div>
                <div class="pstar">{{ mainStars(g) }}</div>
                <div class="pminor">{{ minorStars(g) }}</div>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane label="八宅" name="bazhai">
            <div v-if="bazhai" class="mingua-row">
              <div class="mingua font-serif">{{ bazhai.ming_gua.gua.name }}</div>
              <div>
                <div class="mg-name">{{ bazhai.ming_gua.group }}</div>
                <div class="mg-sub">四吉四凶方</div>
              </div>
            </div>
            <div v-if="bazhai" class="dirs">
              <div v-for="d in bazhaiDirs" :key="d.name" :class="['dir', { good: d.good }]">
                <div class="dg font-serif">{{ d.gua }}</div>
                <div class="dname">{{ d.name }}</div>
              </div>
            </div>
            <template v-if="liuNianXing">
              <div class="sec-title">流年飞星（{{ liuNianXing.year }}）</div>
              <div class="ln-row">入中星：<strong>{{ liuNianXing.ru_zhong }}</strong></div>
              <div class="fly-grid">
                <div
                  v-for="f in flyStars"
                  :key="f.gong_num"
                  :class="['fly-cell', { center: f.gong_num === 5 }]"
                >
                  <div class="fly-name">{{ f.xing_name }}</div>
                </div>
              </div>
            </template>
          </el-tab-pane>
        </el-tabs>
        <div class="correction">{{ correction }}</div>
      </template>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref, watch } from 'vue'
import {
  engineBazhaiChart,
  engineBaziChart,
  engineBaziFullChart,
  engineCityCoords,
  engineSolarTime,
  engineZiweiChart,
  engineZiweiFullChart,
  type BazhaiChart,
  type BaziChartData,
  type BaziSanYuan,
  type BaziZhuFull,
  type BaziYongShen,
  type ZiweiChart,
} from '@/api/engine'
import type { Profile } from '@/api/profiles'
import AppAlert from '@/components/app/AppAlert.vue'

const props = defineProps<{ profile: Profile; auto?: boolean }>()

const loading = ref(false)
const error = ref('')
const correction = ref('')
const tab = ref('bazi')
const baziChart = ref<BaziChartData | null>(null)
const baziFull = ref<{ nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull; san_yuan?: BaziSanYuan } | null>(null)
const ziwei = ref<ZiweiChart | null>(null)
const bazhai = ref<BazhaiChart | null>(null)
const yongShen = ref<BaziYongShen | null>(null)

const anyResult = computed(() => !!(baziChart.value || ziwei.value || bazhai.value))

function zhuText(p?: { gan: string; zhi: string }): string {
  return p ? `${p.gan}${p.zhi}` : ''
}

const baziPillars = computed(() => {
  const src = baziFull.value ?? (baziChart.value as unknown as { nian: BaziZhuFull; yue: BaziZhuFull; ri: BaziZhuFull; shi: BaziZhuFull } | null)
  if (!src) return []
  const map = (label: string, p: BaziZhuFull) => {
    const cang = p.cang_gan ? [p.cang_gan.main, p.cang_gan.mid, p.cang_gan.minor].filter(Boolean).join(' ') : ''
    return {
      label,
      self: label === '日柱',
      gan: p.gan,
      zhi: p.zhi,
      naYin: p.na_yin ?? '',
      shiShen: p.shi_shens?.[0]?.shi_shen ?? '',
      cangGan: cang ? `藏干 ${cang}` : '',
      isVoid: !!p.is_void,
    }
  }
  return [
    map('年柱', src.nian),
    map('月柱', src.yue),
    map('日柱', src.ri),
    map('时柱', src.shi),
  ]
})

const sanYuanText = computed(() => {
  const sy = baziFull.value?.san_yuan
  if (!sy) return ''
  return `胎元 ${zhuText(sy.tai_yuan)} · 命宫 ${zhuText(sy.ming_gong)} · 身宫 ${zhuText(sy.shen_gong)}`
})

const shenShaList = computed(() => {
  const src = baziFull.value
  if (!src) return []
  const names = new Set<string>()
  for (const p of [src.nian, src.yue, src.ri, src.shi]) {
    for (const s of p.shen_sha ?? []) {
      if (s.name) names.add(s.name)
    }
  }
  return [...names]
})

const daYunSteps = computed(() => baziChart.value?.da_yun?.steps ?? [])
const currentStepIndex = computed(() => baziChart.value?.da_yun?.current_step_index ?? -1)

const yongShenSummary = computed(() => {
  const ys = yongShen.value
  if (!ys) return []
  const rows: Array<{ label: string; text: string }> = []
  if (ys.fu_yi?.qiangruo) {
    rows.push({
      label: '扶抑',
      text: `${ys.fu_yi.qiangruo} · 喜 ${ys.fu_yi.xi ?? '—'} 忌 ${ys.fu_yi.ji ?? '—'}`,
    })
  }
  if (ys.tiao_hou?.yong) {
    rows.push({ label: '调候', text: `用 ${ys.tiao_hou.yong} · 喜 ${ys.tiao_hou.xi ?? '—'}` })
  }
  if (ys.ge_ju?.ge_ju) {
    rows.push({
      label: '格局',
      text: `${ys.ge_ju.ge_ju}${ys.ge_ju.yong_fa ? ` · ${ys.ge_ju.yong_fa}` : ''} · 喜 ${ys.ge_ju.xi ?? '—'} 忌 ${ys.ge_ju.ji ?? '—'}`,
    })
  }
  return rows
})

const ziweiGongs = computed(() => ziwei.value?.gong_wei ?? [])
function mainStars(g: { xing_yao: Array<{ name: string; is_zhu_xing?: boolean; liang_du?: string }> }) {
  return g.xing_yao.filter(s => s.is_zhu_xing).map(s => (s.liang_du ? `${s.name}${s.liang_du}` : s.name)).join(' ')
}
function minorStars(g: { xing_yao: Array<{ name: string; is_zhu_xing?: boolean; liang_du?: string }> }) {
  return g.xing_yao.filter(s => !s.is_zhu_xing).map(s => (s.liang_du ? `${s.name}${s.liang_du}` : s.name)).join(' ')
}

const BAZHAI_AUS = ['sheng_qi', 'tian_yi', 'yan_nian', 'fu_wei']
const bazhaiDirs = computed(() => {
  if (!bazhai.value) return []
  const dirs = bazhai.value.ba_zhai_dirs
  return Object.entries(dirs).map(([key, values]) => ({
    name: key, gua: (values ?? [''])[0], good: BAZHAI_AUS.includes(key),
  }))
})

const liuNianXing = computed(() => bazhai.value?.liu_nian_xing ?? null)
const flyStars = computed(() => {
  const gong = liuNianXing.value?.gong_wei
  if (!gong || gong.length !== 9) return []
  // gong_num 1-9：按洛书九宫排 3x3（1下、2右上…按洛书序）
  const order = [4, 9, 2, 3, 5, 7, 8, 1, 6]
  return order.map(n => gong.find(g => g.gong_num === n) ?? { gong_num: n, xing_name: '—' })
})

async function compute() {
  const p = props.profile
  error.value = ''
  correction.value = ''
  baziChart.value = null
  baziFull.value = null
  yongShen.value = null
  ziwei.value = null
  bazhai.value = null
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
    // 真太阳时两路：有地点+精确时刻 → 校正；其余取东八区 120° 仅做历法换算
    let longitude = 120
    if (p.birth_time_known === 'known' && p.birth_place) {
      const coords = await engineCityCoords(p.birth_place)
      longitude = coords.longitude
    }
    const t = await engineSolarTime(datetime, longitude)
    const solar = p.birth_time_known === 'known' && p.birth_place ? t.solar : datetime
    if (p.birth_time_known === 'known' && p.birth_place) correction.value = '已做真太阳时修正'
    else if (p.birth_time_known === 'known') correction.value = '未做真太阳时修正（未填出生地）'
    else if (p.birth_place) correction.value = '按已定时辰排盘（出生地仅用于精确时间校正，此时不生效）'
    else correction.value = '按已定时辰排盘，未做校正'

    baziChart.value = await engineBaziChart(solar, p.gender)
    const full = await engineBaziFullChart(baziChart.value)
    baziFull.value = full
    yongShen.value = full.yong_shen ?? null
    const zwChart = await engineZiweiChart(t.lunar, p.gender)
    ziwei.value = await engineZiweiFullChart(zwChart)
    const birthYear = baziChart.value.birth_year ?? parseInt(p.birth_date.slice(0, 4), 10)
    bazhai.value = await engineBazhaiChart(birthYear, p.gender)
  } catch (err) {
    error.value = toUserMessage(err, '排盘失败')
  } finally {
    loading.value = false
  }
}

// 自动排盘：进入档案详情（或切换档案）即出盘，免手动点「排盘」。
watch(() => props.profile?.id, () => {
  if (props.auto && props.profile) void compute()
}, { immediate: true })
</script>

<style scoped>
.paipan-preview {
  border-radius: 12px; margin-bottom: 20px;
  position: relative;
  border-top: 2px solid var(--liki-gold);
}
.preview-head { display: flex; align-items: center; justify-content: space-between; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); font-size: 13px; margin: 4px 0 12px; }
.pan-skeleton .skel { height: 14px; border-radius: 6px; background: linear-gradient(90deg,#f0efeb 25%,#e7e4de 50%,#f0efeb 75%); background-size: 200% 100%; animation: shimmer 1.2s infinite; margin-bottom: 10px; }
@keyframes shimmer { to { background-position: -200% 0; } }

.pillars {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: 8px; overflow: hidden;
}
.pillar { background: var(--liki-surface); padding: 16px 8px; text-align: center; }
.pillar.self { background: linear-gradient(180deg, var(--liki-gold-tint), #fdfaf3); }
.pillar .label { font-size: 12px; color: var(--liki-text-3); margin-bottom: 8px; letter-spacing: 0.08em; }
.pillar .gl { font-size: 25px; font-weight: 700; letter-spacing: 0.02em; }
.pillar.self .gl { color: #8a6a2f; }
.pillar .sub { font-size: 12px; color: var(--liki-text-2); margin-top: 4px; }
.pillar .nayin { color: var(--liki-text-3); }
.pillar .cang { color: var(--liki-text-3); font-size: 11px; }
.void-tag { display: inline-block; margin-top: 6px; padding: 1px 8px; border-radius: 999px; background: var(--liki-gold-tint); color: #8a6a2f; font-size: 11px; }

.san-yuan, .shensha { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 12px; font-size: 13px; }
.sy-label { color: var(--liki-text-3); flex: 0 0 2em; }
.sy-body { color: var(--liki-text-1); }
.shensha { gap: 6px; }
.ss-chip { padding: 2px 10px; border-radius: 999px; border: 1px solid var(--liki-line); background: var(--liki-surface); color: var(--liki-text-2); font-size: 12px; }

.ln-row { font-size: 13px; color: var(--liki-text-2); margin-bottom: 10px; }
.fly-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; width: min(320px, 100%); }
.fly-cell { background: var(--liki-surface); padding: 10px 4px; text-align: center; }
.fly-cell.center { background: var(--liki-gold-tint); }
.fly-name { font-size: 12px; color: var(--liki-text-2); }

.sec-title { font-size: 13px; font-weight: 600; margin: 16px 0 8px; color: var(--liki-text-1); }
.dayun { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 6px; }
.dy { border: 1px solid var(--liki-line); border-radius: 6px; padding: 8px 4px; text-align: center; }
.dy.cur { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.dy-name { font-size: 15px; font-weight: 700; }
.dy-shen { font-size: 11px; color: var(--liki-text-2); margin-top: 2px; }
.dy-year { font-size: 11px; color: var(--liki-text-3); margin-top: 2px; }
.ys { border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; }
.ys-row { display: flex; gap: 10px; padding: 8px 12px; font-size: 13px; }
.ys-row + .ys-row { border-top: 1px solid var(--liki-line); }
.ys-label { color: var(--liki-text-3); flex: 0 0 3em; }
.ys-body { color: var(--liki-text-1); }

.palaces {
  display: grid; grid-template-columns: repeat(6, 1fr); gap: 1px;
  background: var(--liki-line); border: 1px solid var(--liki-line);
  border-radius: 6px; overflow: hidden;
}
.palace { background: var(--liki-surface); padding: 10px 4px; text-align: center; min-height: 68px; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.palace.ming { background: linear-gradient(180deg, var(--liki-gold-tint), #fdfaf3); }
.palace.shen { background: var(--liki-primary-tint); }
.palace .pname { font-size: 13px; font-weight: 600; }
.palace.ming .pname, .palace.ming .pstar { color: #8a6a2f; }
.palace .pstar { font-size: 12px; color: var(--liki-primary); margin-top: 3px; }
.palace .pminor { font-size: 11px; color: var(--liki-text-3); margin-top: 2px; }

.mingua-row { display: flex; align-items: center; gap: 16px; margin-bottom: 14px; }
.mingua { display: grid; place-items: center; width: 60px; height: 60px; border-radius: 50%; border: 2px solid var(--liki-gold); font-size: 26px; color: #8a6a2f; background: var(--liki-gold-tint); }
.mg-name { font-size: 15px; font-weight: 600; }
.mg-sub { color: var(--liki-text-2); font-size: 12px; margin-top: 2px; }
.dirs { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.dir { border: 1px solid var(--liki-line); border-radius: 6px; padding: 10px; text-align: center; }
.dir .dg { font-size: 18px; color: var(--liki-gold); }
.dir.good { border-color: #bfe6bf; background: #f4faf4; }
.dir.good .dg { color: var(--el-color-success); }
.dir .dname { font-size: 12px; color: var(--liki-text-2); margin-top: 3px; }

.correction { margin-top: 12px; font-size: 12px; color: var(--liki-gold); }
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
  .palaces { grid-template-columns: repeat(3, 1fr); }
  .dirs { grid-template-columns: repeat(2, 1fr); }
}
</style>