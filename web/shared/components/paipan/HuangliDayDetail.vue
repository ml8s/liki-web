<template>
  <div class="d-detail">
    <div class="d-grid">
      <div
        v-if="day.huangdao"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.path') }}</span>
        <span class="d-value">{{ day.huangdao.name }}<template v-if="pathLabel"> · {{ pathLabel }}</template></span>
      </div>
      <div
        v-if="mansionText"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.mansion') }}</span>
        <span class="d-value">{{ mansionText }}</span>
      </div>
      <div
        v-if="naYin"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.naYin') }}</span>
        <span class="d-value">{{ naYin }}</span>
      </div>
      <div
        v-if="spiritsText"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.spirits') }}</span>
        <span class="d-value">{{ spiritsText }}</span>
      </div>
      <div
        v-if="day.jie_qi"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.jieQi') }}</span>
        <span class="d-value">{{ jieQiText }}</span>
      </div>
      <div
        v-if="day.ren_yuan"
        class="d-item"
      >
        <span class="d-label">{{ $t('services.huangli.renYuan') }}</span>
        <span class="d-value">{{ day.ren_yuan }}</span>
      </div>
    </div>
    <div
      v-if="pengZuText"
      class="d-item d-pengzu"
    >
      <span class="d-label">{{ $t('services.huangli.pengZu') }}</span>
      <span class="d-value">{{ pengZuText }}</span>
    </div>
    <div
      v-if="day.shi_chen?.length"
      class="d-shichen"
    >
      <div class="d-label">
        {{ $t('services.huangli.shiChen') }}
      </div>
      <div class="sc-grid">
        <span
          v-for="s in day.shi_chen"
          :key="s.zhi"
          :class="['sc-chip', { good: s.suitable }]"
        >
          <span class="sc-zhi font-serif">{{ s.zhi }}</span>
          <span class="sc-time">{{ s.time }}</span>
          <span class="sc-fu">{{ s.huangdao }} · {{ s.suitable ? $t('services.huangli.auspicious') : $t('services.huangli.inauspicious') }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 黄历单日详情（值神/星宿/纳音/吉神方位/节气/人元/彭祖百忌/十二时辰吉凶）——
// 纯展示、数据驱动；由 HuangliDaysPanel 展开后渲染，App 与营销岛共用。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { HuangliDay } from '../../api/engine/huangli'

const props = defineProps<{ day: HuangliDay }>()
const { t } = useI18n()

// 引擎值 → i18n 键映射；未知值原样透传（传统术语保持中文）。
const PATH_KEYS: Record<string, string> = {
  黄道: 'services.huangli.pathValue.yellow',
  黑道: 'services.huangli.pathValue.black',
}
const DIR_KEYS: Record<string, string> = {
  正东: 'services.huangli.dir.e',
  东: 'services.huangli.dir.e',
  东南: 'services.huangli.dir.se',
  正南: 'services.huangli.dir.s',
  南: 'services.huangli.dir.s',
  西南: 'services.huangli.dir.sw',
  正西: 'services.huangli.dir.w',
  西: 'services.huangli.dir.w',
  西北: 'services.huangli.dir.nw',
  正北: 'services.huangli.dir.n',
  北: 'services.huangli.dir.n',
  东北: 'services.huangli.dir.ne',
}

function mapped(keys: Record<string, string>, value?: string): string {
  if (!value) return ''
  const key = keys[value]
  return key ? t(key) : value
}

const pathLabel = computed(() => mapped(PATH_KEYS, props.day.huangdao?.path))
const naYin = computed(() => props.day.na_yin || props.day.ri_zhu?.na_yin || '')
const mansionText = computed(() => {
  const m = props.day.mansion
  if (!m) return ''
  return [m.name, m.group, m.wuxing].filter(Boolean).join(' · ')
})
const spiritsText = computed(() => {
  const d = props.day
  const parts: string[] = []
  if (d.xi_shen) parts.push(`${t('services.huangli.xiShen')} ${mapped(DIR_KEYS, d.xi_shen)}`)
  if (d.cai_shen) parts.push(`${t('services.huangli.caiShen')} ${mapped(DIR_KEYS, d.cai_shen)}`)
  if (d.fu_shen) parts.push(`${t('services.huangli.fuShen')} ${mapped(DIR_KEYS, d.fu_shen)}`)
  return parts.join(' · ')
})
const jieQiText = computed(() => {
  const days = props.day.jie_qi_days
  return days ? `${props.day.jie_qi} · ${t('services.huangli.jieQiDays', { days })}` : props.day.jie_qi
})
const pengZuText = computed(() => [props.day.gan_ji, props.day.zhi_ji].filter(Boolean).join(' · '))
</script>

<style scoped>
.d-detail { margin-top: var(--space-2); padding-top: var(--space-2); border-top: 1px solid var(--liki-line); text-align: left; }
.d-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: var(--space-1) var(--space-4); }
.d-item { display: flex; flex-direction: column; gap: 0; }
.d-label { font-size: var(--font-size-micro); color: var(--liki-text-3); }
.d-value { font-size: var(--font-size-caption); color: var(--liki-ink); }
.d-pengzu { margin-top: var(--space-2); }
.d-shichen { margin-top: var(--space-2); }
.sc-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: var(--space-1); margin-top: var(--space-1); }
.sc-chip { display: flex; flex-direction: column; align-items: center; gap: 1px; padding: 4px 6px; border: 1px solid var(--liki-line); border-radius: var(--radius-small); text-align: center; }
.sc-chip.good { border-color: var(--liki-primary-border); background: var(--liki-primary-tint); }
.sc-zhi { font-size: var(--font-size-caption); font-weight: 600; }
.sc-chip.good .sc-zhi { color: var(--liki-primary); }
.sc-time { font-size: var(--font-size-micro); color: var(--liki-text-3); font-variant-numeric: tabular-nums; white-space: nowrap; }
.sc-fu { font-size: var(--font-size-micro); color: var(--liki-text-2); }
.sc-chip.good .sc-fu { color: var(--liki-primary); }
</style>
