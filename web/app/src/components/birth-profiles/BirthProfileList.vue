<template>
  <div class="profile-list">
    <BirthProfileCard
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
import type { BirthProfile } from '@/api/birth-profiles'
import BirthProfileCard from '@/components /birth-profiles/BirthProfileCard.vue'

defineProps<{ profiles: BirthProfile[] }>()
defineEmits<{ open: [BirthProfile]; edit: [BirthProfile]; remove: [BirthProfile] }>()
</script>

<style scoped>
.profile-list { display: grid; gap: 12px; }
</style>