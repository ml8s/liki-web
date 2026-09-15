<template>
  <div class="profile-picker">
    <label class="label">咨询对象</label>
    <el-select v-model="selectedId" placeholder="选择一个档案" aria-label="咨询对象" style="width: 240px">
      <el-option v-for="p in profiles" :key="p.id" :value="p.id" :label="p.nickname" />
    </el-select>
    <router-link to="/profiles/new" class="new-link">＋ 新建档案</router-link>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Profile } from '@/api/profiles'

const props = defineProps<{ profiles: Profile[]; modelValue?: string }>()
const emit = defineEmits<{ 'update:modelValue': [string]; change: [Profile] }>()

const selectedId = computed({
  get: () => props.modelValue ?? props.profiles[0]?.id ?? '',
  set: (value: string) => {
    emit('update:modelValue', value)
    const profile = props.profiles.find(p => p.id === value)
    if (profile) emit('change', profile)
  },
})
</script>

<style scoped>
.profile-picker { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.label { font-weight: 600; }
.new-link { display: inline-block; padding: 4px 2px; font-size: 14px; color: var(--liki-primary); text-decoration: none; }
</style>