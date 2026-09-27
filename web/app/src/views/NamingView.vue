<template>
  <section class="service-view">
    <AppPageHeader
      :title="$t('services.naming.title')"
      :subtitle="$t('services.naming.subtitle')"
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
          {{ $t('services.naming.selectProfile') }}
        </h2>
      </template>
      <el-select
        v-model="profileId"
        :aria-label="$t('chat.selectProfile')"
        :placeholder="$t('chat.selectProfile')"
        style="width: 100%"
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
        {{ $t('services.naming.noProfilePrefix') }}<a href="/app/profiles/new">{{ $t('profiles.create') }}</a>{{ $t('services.naming.noProfileSuffix') }}
      </p>
    </el-card>

    <el-card
      v-if="loading"
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.naming.favorableElements') }}
        </h2>
      </template>
      {{ $t('profiles.chart.compute') }}…
    </el-card>

    <template v-if="preview">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.naming.favorableElements') }}
          </h2>
        </template>
        <div class="ys">
          {{ preview.yongText }}
        </div>
        <div class="bazi-info">
          <div
            v-if="preview.wangShuai"
            class="info-row"
          >
            <span class="info-label">{{ $t('services.naming.dayMaster') }}</span><span>{{ preview.wangShuai }}</span>
          </div>
          <div
            v-if="preview.zodiac"
            class="info-row"
          >
            <span class="info-label">{{ $t('services.naming.zodiac') }}</span><span>{{ preview.zodiac }}</span>
          </div>
          <div
            v-if="preview.jiText"
            class="info-row"
          >
            <span class="info-label">{{ $t('services.naming.unfavorable') }}</span><span class="warn-text">{{ preview.jiText }}</span>
          </div>
          <div
            v-if="preview.wuxingStats"
            class="info-row"
          >
            <span class="info-label">{{ $t('services.naming.elementStats') }}</span><span>{{ preview.wuxingStats }}</span>
          </div>
        </div>
        <div
          v-if="singlePool.length"
          class="sec-title"
        >
          {{ $t('services.naming.singleCharacters') }}
        </div>
        <div
          v-if="singlePool.length"
          class="chars"
        >
          <span
            v-for="c in singlePool"
            :key="c.char"
            class="char-item"
          >
            <span class="char">{{ c.char }}</span>
            <span
              v-if="c.frequency"
              class="char-freq"
            >{{ c.frequency }}</span>
          </span>
        </div>
        <div
          v-if="doublePool.length"
          class="sec-title"
        >
          {{ $t('services.naming.doubleCharacters') }}
        </div>
        <div
          v-if="doublePool.length"
          class="chars"
        >
          <span
            v-for="c in doublePool"
            :key="c.char"
            class="char-item"
          >
            <span class="char">{{ c.char }}</span>
            <span
              v-if="c.frequency"
              class="char-freq"
            >{{ c.frequency }}</span>
          </span>
        </div>
      </el-card>

      <AppConsultationCard
        :title="$t('services.naming.consultationTitle')"
        :description="$t('services.naming.consultationBody')"
        :locked-description="$t('services.naming.lockedBody')"
        :purchased="entitlements.purchased"
        :action-label="$t('services.naming.cta')"
        @action="startChat"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import { engineBaziFullChart, engineQimingPick, type BaziYongShen } from '@/api/engine'
import { baziChartFor } from '@/utils/paipan'
import { toUserMessage } from '@/utils/errors'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const router = useRouter()
const { t } = useI18n()
const { profiles, entitlements, chat } = useServiceView()

const profileId = ref('')
const preview = ref<{
  yongText: string
  jiText: string
  wangShuai: string
  zodiac: string
  wuxingStats: string
  singlePool: Array<{ char: string; frequency?: string }>
  doublePool: Array<{ char: string; frequency?: string }>
} | null>(null)
const loading = ref(false)
const error = ref('')

const profile = computed(() => profiles.list.find(p => p.id === profileId.value) ?? null)

const singlePool = computed(() => preview.value?.singlePool ?? [])
const doublePool = computed(() => preview.value?.doublePool ?? [])

function formatWuxingStats(stats?: Array<{ wuxing?: string; count?: number }>): string {
  if (!stats?.length) return ''
  return stats.map(s => `${s.wuxing ?? '?'}:${s.count ?? 0}`).join(' ')
}

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
    if (!yong) throw new Error(t('services.naming.noUsefulGod'))
    const qiangruo = full.yong_shen?.fu_yi?.qiangruo ?? ''
    const yongText = t('services.naming.favorableValue', {
      strength: qiangruo,
      primary: yong,
      secondary: xi && xi !== yong ? `/${xi}` : '',
    })
    const ji = full.yong_shen?.fu_yi?.ji ?? full.yong_shen?.ge_ju?.ji ?? ''
    const jiText = ji ? t('services.naming.unfavorableValue', { value: ji }) : ''
    const wangShuai = qiangruo
    const zodiac = full.zodiac ?? ''
    const wuxingStats = formatWuxingStats(full.element_states)
    const [single, double] = await Promise.all([
      engineQimingPick(yong, xi !== yong ? xi : '', 1),
      engineQimingPick(yong, xi !== yong ? xi : '', 2),
    ])
    const poolChars = (r: typeof single) => (r.pools?.[0]?.chars ?? [])
    preview.value = {
      yongText, jiText, wangShuai, zodiac, wuxingStats,
      singlePool: poolChars(single), doublePool: poolChars(double)
    }
  } catch (err) {
    error.value = toUserMessage(err, t('services.naming.previewFailed'))
  } finally {
    loading.value = false
  }
}

function startChat() {
  const p = profile.value
  if (!p) return
  chat.pendingIntent = t('services.naming.intent')
  void chat.ensureSession(p.id).then(s => { if (s) void router.push(`/chat/${s.id}`) }).catch(err => { error.value = toUserMessage(err, t('chat.createFailed')) })
}
</script>

<style scoped>
.ys { font-size: 15px; color: var(--liki-ink); }
.bazi-info { margin-top: 10px; display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: var(--liki-text-2); }
.info-row { display: flex; gap: 8px; }
.info-label { font-weight: 600; color: var(--liki-ink); min-width: 70px; }
.warn-text { color: var(--el-color-warning); }
.chars { display: flex; flex-wrap: wrap; gap: 8px; }
.char-item { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.char { display: grid; place-items: center; width: 44px; height: 44px; border: 1px solid var(--liki-line); border-radius: 8px; font-size: 20px; font-family: var(--liki-font-serif, serif); background: var(--liki-surface); }
.char-freq { font-size: 11px; color: var(--liki-text-2); }
</style>
