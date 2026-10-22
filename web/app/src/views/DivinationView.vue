<template>
  <ServicePageShell
    :title="$t('services.divination.pageTitle')"
    :subtitle="$t('services.divination.subtitle')"
    :error="error"
  >
    <el-card
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('services.divination.cast') }}
        </h2>
      </template>
      <DivinationCastingForm
        ref="castingFormRef"
        variant="full"
        :loading="loading"
        @submit="cast"
      />
    </el-card>

    <template v-if="snapshot">
      <el-card
        class="app-card app-card--highlight"
        shadow="never"
      >
        <template #header>
          <h2 class="app-card__title">
            {{ $t('services.divination.hexagram') }}
          </h2>
        </template>
        <div class="gua-row">
          <div class="gua-name font-serif">
            {{ board?.name ?? '—' }}
          </div>
          <div class="gua-sub">
            <span v-if="board?.ben_gua">{{ $t('services.divination.original') }} {{ board?.ben_gua }}</span>
            <span v-if="board?.bian_gua"> · {{ $t('services.divination.changed') }} {{ board?.bian_gua }}</span>
          </div>
        </div>
        <div class="gua-info">
          <span v-if="board?.gong">{{ $t('services.divination.palace', { palace: board?.gong }) }}</span>
          <span v-if="board?.gong_wuxing"> · {{ board?.gong_wuxing }}</span>
          <span v-if="board?.ri_chen_gan"> · {{ $t('services.divination.dayPillar') }} {{ board?.ri_chen_gan }}{{ board?.ri_chen_zhi }}</span>
          <span v-if="board?.yue_jian_zhi"> · {{ $t('services.divination.monthBranch') }} {{ board?.yue_jian_gan }}{{ board?.yue_jian_zhi }}</span>
          <span v-if="xunKongText"> · {{ $t('profiles.chart.void') }} {{ xunKongText }}</span>
        </div>

        <div
          v-if="lines.length"
          class="lines-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.lines') }}
          </div>
          <div class="lines-table-wrap">
            <div class="lines-table">
              <div class="line-row line-header">
                <span class="lc">{{ $t('services.divination.position') }}</span>
                <span class="lc">{{ $t('services.divination.relation') }}</span>
                <span class="lc">{{ $t('services.divination.branch') }}</span>
                <span class="lc">{{ $t('services.divination.element') }}</span>
                <span class="lc">{{ $t('services.divination.stem') }}</span>
                <span class="lc">{{ $t('services.divination.strength') }}</span>
                <span class="lc">{{ $t('services.divination.dayRelation') }}</span>
              </div>
              <div
                v-for="(line, idx) in lines"
                :key="idx"
                :class="['line-row', { dong: isDong(idx) }]"
              >
                <span class="lc">{{ $t('services.divination.lineNumber', { number: 6 - idx }) }}</span>
                <span class="lc">{{ line.liu_qin ?? '—' }}</span>
                <span class="lc">{{ line.zhi ?? '—' }}</span>
                <span class="lc">{{ line.wuxing ?? '—' }}</span>
                <span class="lc">{{ line.gan ?? '—' }}</span>
                <span class="lc">{{ wangShuai[idx] ?? '—' }}</span>
                <span class="lc">{{ dayRelations[idx]?.relations?.join('、') ?? '—' }}</span>
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="bianLines.length"
          class="lines-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.changingLines') }}
          </div>
          <div class="lines-table">
            <div
              v-for="(line, idx) in bianLines"
              :key="idx"
              class="line-row"
            >
              <span class="lc">{{ $t('services.divination.lineNumber', { number: 6 - idx }) }}</span>
              <span class="lc">{{ line.liu_qin ?? '—' }}</span>
              <span class="lc">{{ line.zhi ?? '—' }}</span>
              <span class="lc">{{ line.gan ?? '—' }}</span>
            </div>
          </div>
        </div>

        <div
          v-if="focusText"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.usefulGod') }}
          </div>
          <div class="rel-text">
            {{ focusText }}
          </div>
          <div
            v-if="yongShenDetail"
            class="rel-sub"
          >
            {{ yongShenDetail }}
          </div>
        </div>

        <div
          v-if="guaCi"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.hexagramText') }}
          </div>
          <div class="rel-text">
            {{ guaCi }}
          </div>
        </div>

        <div
          v-if="patterns.length"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.patterns') }}
          </div>
          <div
            v-for="p in patterns"
            :key="`${p.type}-${p.sub_type}-${p.position}`"
            class="pattern-item"
          >
            <span class="pattern-name">{{ p.sub_type }}</span>
            <span class="pattern-desc">{{ p.type }} · 爻位 {{ p.position }} · {{ p.is_true ? '有效' : '候选' }}</span>
          </div>
        </div>

        <div
          v-if="timingText"
          class="rel-sec"
        >
          <div class="sec-title">
            {{ $t('services.divination.timing') }}
          </div>
          <div class="rel-text">
            {{ timingText }}
          </div>
        </div>
      </el-card>

      <AppConsultationCard
        :title="$t('services.divination.consultationTitle')"
        :description="$t('services.divination.consultationBody')"
        :locked-description="$t('services.divination.lockedBody')"
        :purchased="entitlements.purchased && profiles.list.length > 0"
        :locked-action-label="profiles.list.length ? $t('commerce.activateNow') : $t('profiles.create')"
        :to="profiles.list.length ? '/chat' : '/birth-profiles/new'"
        @action="startChat"
      />
    </template>

    <el-card
      v-if="toolRecords.length || recordsError"
      class="app-card"
      shadow="never"
    >
      <template #header>
        <h2 class="app-card__title">
          {{ $t('records.title') }}
        </h2>
      </template>
      <RecordList
        :records="toolRecords"
        :error="recordsError"
        @open="openRecord"
        @rename="renameToolRecord"
        @remove="removeToolRecord"
      />
    </el-card>
  </ServicePageShell>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServiceView } from '@/composables/useServiceView'
import {
  divinationCastingFor,
  divinationChartFor,
  type LiuyaoChartResult,
} from '@shared/domain/divination'
import { useToolRecords } from '@/composables/useToolRecords'
import RecordList from '@/components/records/RecordList.vue'
import type { ToolRecord } from '@/api/records'
import AppConsultationCard from '@/components/app/AppConsultationCard.vue'
import ServicePageShell from '@/components/app/ServicePageShell.vue'
import DivinationCastingForm from '@shared/components/divination/DivinationCastingForm.vue'
import type { DivinationCastingFormValue } from '@shared/components/divination/types'

const { t } = useI18n()
const { profiles, entitlements, chat, openServiceChat } = useServiceView()
const {
  records: toolRecords,
  error: recordsError,
  load: loadRecords,
  save: saveToolRecord,
  confirmRename: renameToolRecord,
  confirmRemove: removeToolRecord,
} = useToolRecords('liuyao')
void loadRecords()

const castingFormRef = ref<InstanceType<typeof DivinationCastingForm>>()
const snapshot = ref<LiuyaoChartResult | null>(null)
const loading = ref(false)
const error = ref('')

const board = computed(() => snapshot.value)
const lines = computed(() => snapshot.value?.lines ?? [])
const bianLines = computed(() => snapshot.value?.bian_yao ?? [])
const wangShuai = computed(() => snapshot.value?.wang_shuai ?? [])
const dayRelations = computed(() => snapshot.value?.ri_chen_relations ?? [])
const dongYao = computed(() => snapshot.value?.casting?.dong_yao ?? [])
const patterns = computed(() => snapshot.value?.patterns ?? [])

function isDong(idx: number) {
  const pos = 6 - idx
  return dongYao.value.includes(pos as 1 | 2 | 3 | 4 | 5 | 6)
}

const xunKongText = computed(() => {
  const xk = snapshot.value?.xun_kong
  if (!xk || !xk.length) return ''
  return xk.join(' ')
})

const focusText = computed(() => {
  const ys = snapshot.value?.yong_shen
  if (!ys?.name) return ''
  const position = ys.position && ys.position > 0
    ? t('services.divination.positionValue', { value: ys.position })
    : t('services.divination.hidden')
  return `${ys.name} · ${position}`
})

const yongShenDetail = computed(() => {
  const ys = snapshot.value?.yong_shen
  if (!ys) return ''
  const parts: string[] = []
  if (ys.name) parts.push(ys.name)
  if (ys.position) parts.push(t('services.divination.positionValue', { value: ys.position }))
  return parts.length ? t('services.divination.usefulGodDetail', { value: parts.join(t('common.listSeparator')) }) : ''
})

const guaCi = computed(() => {
  const gc = snapshot.value?.gua_ci
  if (!gc) return ''
  return [gc.name, gc.gua_ci].filter(Boolean).join('：')
})

const timingText = computed(() => {
  const tc = snapshot.value?.timing_candidates
  if (!tc || !tc.length) return ''
  return tc.map(item => [item.window || item.id, item.mechanism, item.trigger_branch].filter(Boolean).join('：')).join('；')
})

async function cast(context: DivinationCastingFormValue) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  const location = context.location
  if (!location) {
    loading.value = false
    return
  }
  const castingInstant = new Date().toISOString()
  snapshot.value = null
  try {
    const { casting } = await divinationCastingFor({ mode: context.castingMode })
    snapshot.value = await divinationChartFor({
      instant: castingInstant,
      timezone: location.timezone,
      longitude: location.longitude,
      casting,
      yongShen: '世爻',
    })
    // 卦不可复现：每次摇卦自动落一条历史记录。
    void saveToolRecord(
      recordTitle(context.question.trim()),
      {
        question: context.question.trim(),
        casting_mode: context.castingMode,
        casting_instant: castingInstant,
        location,
      },
      snapshot.value as unknown as Record<string, unknown>,
    )
  } catch (err) {
    error.value = toUserMessage(err, t('services.divination.failed'))
  } finally {
    loading.value = false
  }
}

function recordTitle(question: string): string {
  return t('records.titleLiuyao', { question: question.slice(0, 30) })
}

function openRecord(record: ToolRecord) {
  castingFormRef.value?.restore(record.input)
  snapshot.value = (record.output as unknown as LiuyaoChartResult | undefined) ?? null
}

function startChat() {
  const profile = profiles.list[0]
  if (!profile) return
  const question = castingFormRef.value?.getContext().question.trim() ?? ''
  chat.pendingIntent = t('services.divination.intent', { question })
  void openServiceChat(profile.id, 'divination', message => { error.value = message })
}
</script>

<style scoped>
.gua-row { display: flex; align-items: baseline; gap: var(--space-3); }
.gua-name { font-size: var(--font-size-display-sm); font-weight: 700; }
.gua-sub { color: var(--liki-text-2); font-size: var(--font-size-label); }
.gua-info { margin-top: var(--space-2); font-size: var(--font-size-label); color: var(--liki-text-2); }
/* 区块间距由全局 .service-view .sec-title 的 margin 统一提供，避免双重节奏。 */
.lines-table-wrap { overflow-x: auto; -webkit-overflow-scrolling: touch; }
.lines-table { border: 1px solid var(--liki-line); border-radius: var(--radius-small); overflow: hidden; font-size: var(--font-size-caption); min-width: 480px; }
.line-row { display: flex; }
.line-row + .line-row { border-top: 1px solid var(--liki-line); }
.line-header { background: var(--liki-surface); font-weight: 600; }
.line-row .lc { flex: 1; padding: 6px 4px; text-align: center; }
.line-row.dong { background: var(--liki-primary-tint); }
.rel-text { font-size: var(--font-size-body); color: var(--liki-ink); }
.rel-sub { font-size: var(--font-size-label); color: var(--liki-text-2); margin-top: var(--space-1); }
.pattern-item { display: flex; flex-direction: column; gap: 2px; margin-bottom: var(--space-1); }
.pattern-name { font-weight: 600; color: var(--liki-primary); }
.pattern-desc { font-size: var(--font-size-caption); color: var(--liki-text-3); }
</style>
