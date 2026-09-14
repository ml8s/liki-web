<template>
  <div class="system-profile">
    <div class="head">
      <span class="title font-serif">已载入档案</span>
      <span class="pill">{{ nickname }}</span>
    </div>
    <div class="meta">{{ summary }}</div>
    <p class="note">将基于以上出生信息排盘后解读。</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface ProfileSummary {
  nickname?: string
  name?: string
  gender?: string
  relation?: string
  calendar?: string
  birth_date?: string
  birth_time?: string
  birth_time_known?: string
  birth_place?: string
}

const props = defineProps<{ content: string }>()

const data = computed<ProfileSummary>(() => {
  try {
    return JSON.parse(props.content) as ProfileSummary
  } catch {
    return {}
  }
})

const nickname = computed(() => data.value.nickname ?? '档案')

const summary = computed(() => {
  const d = data.value
  const calendar = d.calendar === 'lunar' ? '农历' : '公历'
  const gender = d.gender === 'male' ? '男' : d.gender === 'female' ? '女' : ''
  const known = { known: '已知', approximate: '时辰', unknown: '不确定' }[d.birth_time_known ?? ''] ?? ''
  const place = d.birth_place ? ` · ${d.birth_place}` : ''
  return [calendar, d.birth_date, d.birth_time, gender, known, place].filter(Boolean).join(' ')
})
</script>

<style scoped>
.system-profile {
  border: 1px solid var(--liki-line); border-top: 2px solid var(--liki-gold);
  border-radius: 12px; background: var(--liki-surface); padding: 14px 18px;
}
.head { display: flex; align-items: center; gap: 8px; }
.title { font-size: 15px; font-weight: 700; }
.pill { font-size: 11px; background: var(--liki-gold-tint); color: #8a6a2f; padding: 1px 8px; border-radius: 999px; }
.meta { color: var(--liki-text-2); font-size: 13px; margin-top: 8px; }
.note { color: var(--liki-text-3); font-size: 12px; margin: 6px 0 0; }
</style>