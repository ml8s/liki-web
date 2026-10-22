<template>
  <div class="system-profile">
    <div class="head">
      <span class="title font-serif">{{ $t('chat.systemBirthProfile.title') }}</span>
      <span class="pill">{{ nickname }}</span>
    </div>
    <div class="meta">
      {{ summary }}
    </div>
    <p class="note">
      {{ $t('chat.systemBirthProfile.note') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BirthProfile } from '@/api/birth-profiles'
import { formatInstant } from '@shared/domain/time'

const props = defineProps<{ content: string }>()
const { t } = useI18n()

const data = computed<Partial<BirthProfile>>(() => {
  try {
    return JSON.parse(props.content) as Partial<BirthProfile>
  } catch {
    return {}
  }
})

const nickname = computed(() => data.value.nickname ?? t('profiles.profile'))

const summary = computed(() => {
  const d = data.value
  const gender = d.birth?.gender === 'male' ? t('profiles.male') : d.birth?.gender === 'female' ? t('profiles.female') : ''
  const place = d.birth?.location?.place ? ` · ${d.birth.location.place}` : ''
  const canonicalTime = d.birth?.instant && d.birth?.location?.timezone
    ? formatInstant(d.birth.instant, d.birth.location.timezone)
    : [d.birth?.date, d.birth?.time].filter(Boolean).join(' ')
  return [canonicalTime, gender, place].filter(Boolean).join(' ')
})
</script>

<style scoped>
.system-profile {
  border: 1px solid var(--liki-line); border-top: 2px solid var(--liki-primary);
  border-radius: var(--radius-lg); background: var(--liki-surface); padding: 14px 18px;
}
.head { display: flex; align-items: center; gap: 8px; }
.title { font-size: var(--font-size-body-lg); font-weight: 700; }
.pill { font-size: var(--font-size-micro); background: var(--liki-primary-tint); color: var(--liki-primary); padding: 1px 8px; border-radius: var(--radius-full); }
.meta { color: var(--liki-text-2); font-size: var(--font-size-label); margin-top: 8px; }
.note { color: var(--liki-text-3); font-size: var(--font-size-caption); margin: 6px 0 0; }
</style>
