<template>
  <div class="profile-picker">
    <label class="label">{{ $t('chat.profilePicker.label') }}</label>
    <el-select
      v-model="selectedId"
      :placeholder="$t('chat.profilePicker.placeholder')"
      :aria-label="$t('chat.profilePicker.label')"
      class="profile-picker__select"
    >
      <el-option
        v-for="p in profiles"
        :key="p.id"
        :value="p.id"
        :label="p.nickname"
      />
    </el-select>
    <router-link
      to=" /birth-profiles/new"
      class="new-link"
    >
      ＋ {{ $t('profiles.create') }}
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { BirthProfile } from '@/api/birth-profiles'

const props = defineProps<{ profiles: BirthProfile[]; modelValue?: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()

const selectedId = computed({
  get: () => props.modelValue ?? props.profiles[0]?.id ?? '',
  set: (value: string) => {
    emit('update:modelValue', value)
  },
})
</script>

<style scoped>
.profile-picker { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.label { font-weight: 600; }
.profile-picker__select { width: 240px; }
.new-link { display: inline-block; padding: 4px 2px; font-size: 14px; color: var(--liki-primary); text-decoration: none; }
</style>
