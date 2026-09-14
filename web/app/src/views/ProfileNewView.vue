<template>
  <section class="new-profile">
    <AppPageHeader title="新建档案" subtitle="保存一个人的出生信息，之后可免费预览排盘或付费解读。" />
    <AppAlert v-if="error" :title="error" type="error" />
    <BirthForm :submitting="submitting" @submit="submit" />
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useProfilesStore } from '@/stores/profiles'
import type { ProfileInput } from '@/api/profiles'
import AppAlert from '@/components/app/AppAlert.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import BirthForm from '@/components/profiles/BirthForm.vue'

const router = useRouter()
const profiles = useProfilesStore()
const submitting = ref(false)
const error = ref('')

async function submit(input: ProfileInput) {
  submitting.value = true
  error.value = ''
  try {
    const profile = await profiles.create(input)
    await router.push(`/profiles/${profile.id}`)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.new-profile { max-width: 560px; }
</style>