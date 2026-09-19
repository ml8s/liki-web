<template>
  <div class="profile-card">
    <div class="avatar font-serif" @click="$emit('open', profile)">{{ initial }}</div>
    <div class="info" @click="$emit('open', profile)">
      <div class="name-row">
        <span class="name font-serif">{{ profile.nickname }}</span>
      </div>
      <div class="meta">{{ birthSummary(profile) }}</div>
    </div>
    <div class="actions">
      <el-button size="small" text type="primary" @click="$emit('open', profile)">查看</el-button>
      <el-button size="small" text @click="$emit('edit', profile)">编辑</el-button>
      <el-button size="small" text type="danger" @click="$emit('remove', profile)">删除</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { birthSummary, type Profile } from '@/api/profiles'

const props = defineProps<{ profile: Profile }>()
defineEmits<{ open: [Profile]; edit: [Profile]; remove: [Profile] }>()

const initial = computed(() => props.profile.nickname.slice(0, 1))
</script>

<style scoped>
.profile-card {
  display: flex; align-items: center; gap: 14px;
  background: var(--liki-surface); border: 1px solid var(--liki-line);
  border-radius: 12px; padding: 16px 18px; cursor: pointer;
  transition: border-color var(--liki-dur) var(--liki-ease), box-shadow var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease);
}
.profile-card:hover {
  border-color: var(--liki-primary-border);
  box-shadow: 0 6px 18px rgba(30, 34, 48, 0.08);
  transform: translateY(-1px);
}
.avatar {
  width: 44px; height: 44px; border-radius: 50%; flex-shrink: 0;
  background: var(--liki-primary-tint); color: var(--liki-primary);
  display: grid; place-items: center; font-size: 18px;
}
.info { flex: 1; min-width: 0; }
.name-row { display: flex; align-items: center; gap: 8px; }
.name { font-size: 17px; font-weight: 700; }
.meta { color: var(--liki-text-2); font-size: 13px; margin-top: 3px; }
.actions { display: flex; gap: 4px; }
@media (max-width: 640px) { .actions { flex-direction: column; } }
</style>