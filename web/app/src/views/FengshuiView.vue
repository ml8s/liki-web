<template>
  <section class="service-view">
    <AppPageHeader
      :title="$t('services.fengshui.pageTitle')"
      :subtitle="$t('services.fengshui.subtitle')"
    />

    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.fengshui.selectBirthProfile') }}
        </h2>
      </template>
      <el-select
        v-model="profileId"
        :aria-label="$t('chat.selectBirthProfile')"
        :placeholder="$t('chat.selectBirthProfile')"
        class="w-full"
        @change="profileId && loadPreview()"
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
        {{ $t('services.fengshui.noBirthProfilePrefix') }}<router-link to="/birth-profiles/new">
          {{ $t('profiles.create') }}
        </router-link>{{ $t('services.fengshui.noBirthProfileSuffix') }}
      </p>
    </el-card>

    <el-card
      v-if="loading"
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.fengshui.lifeGua') }}
        </h2>
      </template>
      {{ $t('profiles.chart.compute') }}…
    </el-card>

    <template v-if="bazhai">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.fengshui.lifeGuaDirections') }}
          </h2>
        </template>
        <div class="mingua-row">
          <div class="mingua font-serif">
            {{ bazhai.ming_gua.gua.name }}
          </div>
          <div>
            <div class="mg-name">
              {{ bazhai.ming_gua.group }}
            </div>
            <div
              v-if="bazhai.ming_gua.gua.wuxing"
              class="mg-wuxing"
            >
              {{ $t('services.fengshui.element') }}{{ bazhai.ming_gua.gua.wuxing }}
            </div>
          </div>
        </div>
        <div class="dirs">
          <div
            v-for="d in bazhaiDirs"
            :key="d.key"
            :class="['dir', { good: d.good }]"
          >
            <div class="dg font-serif">
              {{ d.gua }}
            </div>
            <div class="dname">
              {{ $t(`profiles.chart.direction.${d.key}`) }}
            </div>
          </div>
        </div>
        <template v-if="liuNianXing">
          <div class="sec-title">
            {{ $t('profiles.chart.annualStars', { year: liuNianXing.year }) }}
          </div>
          <div class="ln-row">
            {{ $t('profiles.chart.centerStar') }}<strong>{{ liuNianXing.ru_zhong }}</strong>
          </div>
          <div class="fly-grid">
            <div
              v-for="f in flyStars"
              :key="f.gong_num"
              :class="['fly-cell', { center: f.gong_num === 5, ru_zhong: f.ru_zhong }]"
            >
              <div class="fly-name">
                {{ f.xing_name }}
              </div>
              <div
                v-if="f.wuxing"
                class="fly-wuxing"
              >
                {{ f.wuxing }}
              </div>
            </div>
          </div>
        </template>
      </el-card>

      <AppConsultationCard
        :title="$t('services.fengshui.consultationTitle')"
        :description="$t('services.fengshui.consultationBody')"
        :locked-description="$t('services.fengshui.lockedBody')"
        :purchased="entitlements.purchased"
        :action-label="$t('services.fengshui.consultationTitle')"
        @action="startChat"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { engineBazhaiChart, type BazhaiChart } from '@/api/engine'
import { BAZHAI_AUS, LUOSHU_ORDER } from '@/utils/bazhai'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()

const profileId = ref('')
const bazhai = ref<BazhaiChart | null>(null)
const loading = ref(false)
const error = ref('')

const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

const bazhaiDirs = computed(() => {
  if (!bazhai.value) return []
  return Object.entries(bazhai.value.ba_zhai_dirs).map(([key, values]) => ({
    key, gua: (values ?? [''])[0], good: BAZHAI_AUS.includes(key),
  }))
})
const liuNianXing = computed(() => bazhai.value?.liu_nian_xing ?? null)
const flyStars = computed(() => {
  const gong = liuNianXing.value?.gong_wei
  if (!gong || gong.length !== 9) return []
  return LUOSHU_ORDER.map(n => gong.find(g => g.gong_num === n) ?? { gong_num: n, xing_name: '—' })
})

async function loadPreview() {
  const p = profile.value
  if (!p) return
  loading.value = true
  error.value = ''
  bazhai.value = null
  try {
    if (p.calendar === 'lunar') throw new Error(t('profiles.chart.lunarHint'))
    const birthYear = parseInt(p.birth_date.slice(0, 4), 10)
    if (isNaN(birthYear) || birthYear < 1900 || birthYear > 2100) throw new Error(t('services.fengshui.invalidBirthYear'))
    bazhai.value = await engineBazhaiChart(birthYear, p.gender)
  } catch (err) {
    error.value = toUserMessage(err, t('services.fengshui.failed'))
  } finally {
    loading.value = false
  }
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = t('services.fengshui.intent')
  void openServiceChat(p.id, message => { error.value = message })
}
</script>

<style scoped>
.mingua-row { display: flex; align-items: center; gap: 14px; margin-bottom: 14px; }
.mingua { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 50%; border: 2px solid var(--liki-primary); font-size: 24px; color: var(--liki-primary); background: var(--liki-primary-tint); }
.mg-name { font-size: 15px; font-weight: 600; }
.mg-wuxing { font-size: 13px; color: var(--liki-text-2); margin-top: 2px; }
.dirs { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.dir { border: 1px solid var(--liki-line); border-radius: 6px; padding: 10px; text-align: center; }
.dir .dg { font-size: 16px; color: var(--liki-primary); }
.dir.good { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.dir.good .dg { color: var(--liki-primary); }
.dir .dname { font-size: 12px; color: var(--liki-text-2); margin-top: 3px; }
.ln-row { font-size: 13px; color: var(--liki-text-2); margin-bottom: 10px; }
.fly-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--liki-line); border: 1px solid var(--liki-line); border-radius: 6px; overflow: hidden; width: min(240px, 100%); }
.fly-cell { background: var(--liki-surface); padding: 8px 4px; text-align: center; }
.fly-cell.center { background: var(--liki-primary-tint); }
.fly-cell.ru_zhong { border: 2px solid var(--liki-primary); background: var(--liki-primary-tint); }
.fly-name { font-size: 12px; color: var(--liki-text-2); }
.fly-wuxing { font-size: 11px; color: var(--liki-text-3); margin-top: 2px; }
@media (max-width: 640px) { .dirs { grid-template-columns: repeat(2, 1fr); } }
</style>
