<template>
  <div class="profile-list">
    <ProfileCard
      v-for="profile in profiles"
      :key="profile.id"
      :profile="profile"
      @open="$emit('open', $event)"
      @edit="$emit('edit', $event)"
      @remove="$emit('remove', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { Profile } from '@/api/profiles'
import ProfileCard from '@/components/profiles/ProfileCard.vue'

defineProps<{ profiles: Profile[] }>()
defineEmits<{ open: [Profile]; edit: [Profile]; remove: [Profile] }>()
</script>

<style scoped>
.profile-list { display: grid; gap: 12px; }
</style>