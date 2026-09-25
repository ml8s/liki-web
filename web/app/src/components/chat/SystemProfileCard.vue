<template>
  <div class="system-profile">
    <div class="head">
      <span class="title font-serif">{{ $t('chat.systemProfile.title') }}</span>
      <span class="pill">{{ nickname }}</span>
    </div>
    <div class="meta">{{ summary }}</div>
    <p class="note">{{ $t('chat.systemProfile.note') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Profile } from '@/api/profiles'

const props = defineProps<{ content: string }>()
const { t } = useI18n()

const data = computed<Partial<Profile>>(() => {
  try {
    return JSON.parse(props.content) as Partial<Profile>
  } catch {
    return {}
  }
})

const nickname = computed(() => data.value.nickname ?? t('profiles.profile'))

const summary = computed(() => {
  const d = data.value
  const calendar = d.calendar === 'lunar' ? t('profiles.form.lunar') : t('profiles.form.solar')
  const gender = d.gender === 'male' ? t('profiles.male') : d.gender === 'female' ? t('profiles.female') : ''
  const knownMap: Record<string, string> = {
    known: t('chat.systemProfile.known'),
    approximate: t('chat.systemProfile.approximate'),
    unknown: t('chat.systemProfile.unknown'),
  }
  const known = knownMap[d.birth_time_known ?? ''] ?? ''
  const place = d.birth_place ? ` · ${d.birth_place}` : ''
  return [calendar, d.birth_date, d.birth_time, gender, known, place].filter(Boolean).join(' ')
})
</script>

<style scoped>
.system-profile {
  border: 1px solid var(--liki-line); border-top: 2px solid var(--liki-primary);
  border-radius: 12px; background: var(--liki-surface); padding: 14px 18px;
}
.head { display: flex; align-items: center; gap: 8px; }
.title { font-size: 15px; font-weight: 700; }
.pill { font-size: 11px; background: var(--liki-primary-tint); color: var(--liki-primary); padding: 1px 8px; border-radius: 999px; }
.meta { color: var(--liki-text-2); font-size: 13px; margin-top: 8px; }
.note { color: var(--liki-text-3); font-size: 12px; margin: 6px 0 0; }
</style>
