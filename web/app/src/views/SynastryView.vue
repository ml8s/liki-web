<template>
  <ServicePageShell
    :title="$t('services.bond.title')"
    :subtitle="$t('services.bond.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.bond.step1') }}
        </h2>
      </template>
      <el-select
        v-model="profileAId"
        :aria-label="$t('chat.selectBirthProfile')"
        :placeholder="$t('chat.selectBirthProfile')"
        class="w-full"
        @change="profileAId && loadA()"
      >
        <el-option
          v-for="p in profiles.list"
          :key="p.id"
          :value="p.id"
          :label="p.nickname"
        />
      </el-select>
      <p
        v-if="!profiles.list.length"
        class="hint"
      >
        {{ $t('services.bond.noBirthProfilePrefix') }}<router-link to="/birth-profiles/new">
          {{ $t('profiles.create') }}
        </router-link>{{ $t('services.bond.noBirthProfileSuffix') }}
      </p>
    </el-card>

    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.bond.step2') }}
        </h2>
      </template>
      <el-form
        label-position="top"
        class="b-form"
        @submit.prevent="compute"
      >
        <div class="row">
          <el-form-item :label="$t('profiles.form.gender')">
            <el-radio-group v-model="b.gender">
              <el-radio-button value="female">
                {{ $t('profiles.female') }}
              </el-radio-button>
              <el-radio-button value="male">
                {{ $t('profiles.male') }}
              </el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item :label="$t('profiles.form.birthDate')">
            <el-date-picker
              v-model="b.birth_date"
              type="date"
              value-format="YYYY-MM-DD"
              :placeholder="$t('profiles.form.selectDate')"
              class="w-full"
            />
          </el-form-item>
          <el-form-item :label="$t('profiles.form.birthTime')">
            <el-time-picker
              v-model="b.birth_time"
              format="HH:mm"
              value-format="HH:mm"
              :placeholder="$t('profiles.form.timePlaceholder')"
              class="w-full"
            />
          </el-form-item>
          <el-form-item :label="$t('services.bond.birthPlaceOptional')">
            <el-input
              v-model="b.birth_place"
              name="partner-birth-place"
              autocomplete="off"
              :placeholder="$t('services.bond.birthPlacePlaceholder')"
            />
          </el-form-item>
        </div>
        <el-button
          type="primary"
          native-type="submit"
          :loading="loading"
          :disabled="!profileAId || !b.birth_date || !b.birth_time"
        >
          {{ $t('services.bond.compute') }}
        </el-button>
      </el-form>
    </el-card>

    <template v-if="result">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.bond.result') }}
          </h2>
        </template>

        <div class="charts">
          <div class="chart-row">
            <div class="person">
              {{ aName }}
            </div>
            <div class="pillars">
              <div
                v-for="p in pillarsA"
                :key="p.label"
                class="pillar"
              >
                <div class="pl">
                  {{ p.label }}
                </div>
                <div class="gl font-serif">
                  {{ p.gan }}{{ p.zhi }}
                </div>
                <div class="sub">
                  {{ p.naYin }}
                </div>
              </div>
            </div>
          </div>
          <div class="chart-row">
            <div class="person">
              {{ $t('services.bond.otherPerson') }}
            </div>
            <div class="pillars">
              <div
                v-for="p in pillarsB"
                :key="p.label"
                class="pillar"
              >
                <div class="pl">
                  {{ p.label }}
                </div>
                <div class="gl font-serif">
                  {{ p.gan }}{{ p.zhi }}
                </div>
                <div class="sub">
                  {{ p.naYin }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="dayMasterRel"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.bond.dayMasterRelation') }}
          </div>
          <div class="rel-text">
            {{ dayMasterRel }}
          </div>
        </div>

        <div
          v-if="keyRels.length"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.bond.keyEarthlyBranches') }}
          </div>
          <div class="rels">
            <span
              v-for="r in keyRels"
              :key="r"
              class="rel-chip"
            >{{ r }}</span>
          </div>
        </div>

        <div
          v-if="nayinRels.length"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.bond.nayinRelation') }}
          </div>
          <div class="rels">
            <span
              v-for="r in nayinRels"
              :key="r"
              class="rel-chip nayin"
            >{{ r }}</span>
          </div>
        </div>

        <div
          v-if="shenshaRels.length"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.bond.shenShaCross') }}
          </div>
          <div class="rels">
            <span
              v-for="r in shenshaRels"
              :key="r"
              class="rel-chip shensha"
            >{{ r }}</span>
          </div>
        </div>

        <div
          v-if="shiShenRows.length"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.bond.shiShenCross') }}
          </div>
          <div class="rels">
            <span
              v-for="r in shiShenRows"
              :key="r"
              class="rel-chip"
            >{{ r }}</span>
          </div>
        </div>
      </el-card>

      <AppConsultationCard
        :title="$t('services.bond.consultationTitle')"
        :description="$t('services.bond.consultationBody')"
        :locked-description="$t('services.bond.lockedBody')"
        :purchased="purchased"
        :action-label="$t('services.bond.cta')"
        :locked-action-label="$t('commerce.activateNow')"
        to="/chat"
        @action="startInterpretation"
      />
    </template>
  </ServicePageShell>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import {
  engineBaziBond,
  type BaziBondResult,
  type BaziChartData,
} from '@/api/engine'
import { baziChartFor } from '@/utils/paipan'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'

const { t, te } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()

interface SynastryInput { gender: string; birth_date: string; birth_time: string; birth_place: string }

const profileAId = ref('')
const chartA = ref<BaziChartData | null>(null)
const bChart = ref<BaziChartData | null>(null)
const b = ref<SynastryInput>({ gender: 'female', birth_date: '', birth_time: '', birth_place: '' })
const result = ref<BaziBondResult | null>(null)
const loading = ref(false)
const error = ref('')

const purchased = computed(() => entitlements.allowedSkills.length > 0)
const profileA = computed(() => profiles.list.find(p => p.id === profileAId.value) ?? null)
const aName = computed(() => profileA.value?.nickname ?? t('services.bond.profileA'))

const PILLAR_LABELS = computed(() => [
  t('profiles.chart.pillar.year'),
  t('profiles.chart.pillar.month'),
  t('profiles.chart.pillar.day'),
  t('profiles.chart.pillar.hour'),
])
function pillarsOf(chart: BaziChartData | null) {
  if (!chart) return []
  const list = [chart.nian, chart.yue, chart.ri, chart.shi]
    return PILLAR_LABELS.value.map((label, i) => ({
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
    chartA.value = await baziChartFor(p)
  } catch (err) {
    error.value = toUserMessage(err, t('profiles.detail.loadFailed'))
  }
}

async function compute() {
  error.value = ''
  result.value = null
  bChart.value = null
  if (!profileAId.value) return
  if (!b.value.birth_date || !b.value.birth_time) {
    error.value = t('services.bond.birthRequired')
    return
  }
  loading.value = true
  try {
    if (!chartA.value) await loadA()
    if (!chartA.value) return
    bChart.value = await baziChartFor({ gender: b.value.gender, birth_date: b.value.birth_date, birth_time: b.value.birth_time, birth_time_known: 'known', birth_place: b.value.birth_place })
    result.value = await engineBaziBond(chartA.value, bChart.value)
  } catch (err) {
    error.value = toUserMessage(err, t('services.bond.failed'))
  } finally {
    loading.value = false
  }
}

const ZHU_LABELS = computed<Record<string, string>>(() => ({
  nian: t('profiles.chart.pillar.year'),
  yue: t('profiles.chart.pillar.month'),
  ri: t('profiles.chart.pillar.day'),
  shi: t('profiles.chart.pillar.hour'),
}))

function relText(rel: { relation?: string; detail?: string; type?: string }): string {
  return rel.relation || rel.detail || rel.type || ''
}

const dayMasterRel = computed(() => {
  const pairs = result.value?.zhu_cross?.pairs ?? []
  const pair = pairs.find(p => p.a_zhu === 'ri' && p.b_zhu === 'ri')
  if (!pair) return ''
  return t('services.bond.dayMasterValue', {
    a: aName.value,
    aPillar: `${pair.a_gan}${pair.a_zhi}`,
    bPillar: `${pair.b_gan}${pair.b_zhi}`,
    gan: relText(pair.gan_guan_xi),
    zhi: relText(pair.zhi_guan_xi),
  })
})

const keyRels = computed(() => {
  const pairs = result.value?.zhu_cross?.pairs ?? []
  const set = new Set<string>()
  for (const p of pairs) {
    // 过滤按 type（六合/三合/六冲/六害/相刑），detail 只做展示（如「辰酉合化金」）。
    if (/六合|三合|六冲|六害|相刑/.test(p.zhi_guan_xi.type)) {
      set.add(`${ZHU_LABELS.value[p.a_zhu] ?? p.a_zhu}·${p.a_gan}${p.a_zhi} — ${ZHU_LABELS.value[p.b_zhu] ?? p.b_zhu}·${p.b_gan}${p.b_zhi}：${relText(p.zhi_guan_xi)}`)
    }
  }
  return [...set].slice(0, 8)
})

const nayinRels = computed(() => {
  const pairs = result.value?.nayin_cross?.pairs ?? []
  return pairs.filter(p => p.relation && p.relation !== '无').map(p =>
    `${ZHU_LABELS.value[p.a_zhu] ?? p.a_zhu}·${p.a_na_yin} ↔ ${ZHU_LABELS.value[p.b_zhu] ?? p.b_zhu}·${p.b_na_yin}：${p.relation}`
  )
})

const shenshaRels = computed(() => {
  const cross = result.value?.shensha_cross
  if (!cross) return []
  const rows: string[] = []
  for (const [name, info] of Object.entries(cross)) {
    // 引擎键是固定标识符（tian_yi/kui_gang…），展示走 locale 标签。
    const key = `services.bond.shensha.${name}`
    const label = te(key) ? t(key) : name
    if (info.a_present_in_b) rows.push(t('services.bond.aStarIntoB', { name: label, a: aName.value }))
    if (info.b_present_in_a) rows.push(t('services.bond.bStarIntoA', { name: label, a: aName.value }))
  }
  return rows.slice(0, 10)
})

const shiShenRows = computed(() => {
  const cross = result.value?.shi_shen_cross
  if (!cross) return []
  const rows: string[] = []
  const short: Record<string, string> = {
    nian: t('services.bond.pillarShort.year'),
    yue: t('services.bond.pillarShort.month'),
    ri: t('services.bond.pillarShort.day'),
    shi: t('services.bond.pillarShort.hour'),
  }
  for (const k of ['nian_gan', 'yue_gan', 'ri_gan', 'shi_gan']) {
    const a = cross.a_to_b?.[k]
    const bb = cross.b_to_a?.[k]
    const label = short[k.replace('_gan', '')] ?? k.replace('_gan', '')
    if (a) rows.push(t('services.bond.aViewB', { a: aName.value, pillar: label, value: a }))
    if (bb) rows.push(t('services.bond.bViewA', { a: aName.value, pillar: label, value: bb }))
  }
  return rows
})

async function startInterpretation() {
  const p = profileA.value
  if (!p) return
  const bBirthProfile = b.value
  const bInfo = t('services.bond.otherBirthProfileSummary', {
    gender: bBirthProfile.gender === 'male' ? t('profiles.male') : t('profiles.female'),
    date: bBirthProfile.birth_date,
    time: bBirthProfile.birth_time,
    place: bBirthProfile.birth_place ? t('services.bond.otherPlace', { place: bBirthProfile.birth_place }) : '',
  })
  chat.pendingIntent = t('services.bond.intent', { a: p.nickname, info: bInfo })
  await openServiceChat(p.id, 'natal', message => { error.value = message })
}
</script>

<style scoped>
.row { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; }
.charts { display: grid; gap: 14px; }
.chart-row { display: flex; align-items: center; gap: 12px; }
.person { flex: 0 0 56px; font-weight: 600; font-size: var(--font-size-body); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pillars { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px; flex: 1; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; }
.pillar { background: var(--liki-surface); padding: 8px 4px; text-align: center; }
.pillar .pl { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.pillar .gl { font-size: var(--font-size-title-xs); font-weight: 700; }
.pillar .sub { font-size: var(--font-size-micro); color: var(--liki-text-2); }
.rel-sec { margin-top: 16px; }
.sec-title { font-size: var(--font-size-label); font-weight: 600; margin-bottom: 8px; }
.rel-text { font-size: var(--font-size-body); color: var(--liki-ink); }
.rels { display: flex; flex-wrap: wrap; gap: 8px; }
.rel-chip { padding: 4px 10px; border-radius: var(--radius-full); border: 1px solid var(--liki-line); background: var(--liki-surface); font-size: var(--font-size-caption); color: var(--liki-text-2); }
.rel-chip.nayin { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.rel-chip.shensha { border-color: var(--liki-primary-border, var(--liki-primary-border)); background: var(--liki-primary-tint); }
@media (max-width: 640px) {
  .pillars { grid-template-columns: repeat(2, 1fr); }
  .chart-row { flex-direction: column; align-items: flex-start; gap: 6px; }
  .person { flex: none; }
}
</style>
