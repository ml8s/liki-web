<template>
  <section class="new-profile">
    <AppPageHeader title="新建档案" subtitle="保存一个人的出生信息，之后可免费预览排盘或付费解读。" />
    <AppAlert v-if="draftLoaded" title="已带入主页的出生信息，可继续排盘" type="info" />
    <AppAlert v-if="error" :title="error" type="error" />
    <BirthForm :submitting="submitting" :initial="draft ?? undefined" @submit="submit" />
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
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
const draft = ref<Partial<ProfileInput> | null>(null)
const draftLoaded = ref(false)

// 主页免费排盘钩子（草稿续算）：同步读 sessionStorage 预填出生信息，读后清除。
// 必须在 BirthForm setup 前读到（其内部 ref 从 initial 初始化一次）。
try {
  const raw = sessionStorage.getItem('liki-draft')
  if (raw) {
    const parsed = JSON.parse(raw)
    if (parsed && parsed.birth_date) {
      draft.value = {
        birth_date: parsed.birth_date,
        birth_time: parsed.birth_time ?? '',
        gender: parsed.gender ?? 'female',
        calendar: parsed.calendar ?? 'solar',
        birth_time_known: parsed.birth_time ? 'known' : 'unknown',
        nickname: '',
        third_party_consent: false,
      }
      draftLoaded.value = true
      sessionStorage.removeItem('liki-draft')
    }
  }
} catch {
  sessionStorage.removeItem('liki-draft')
}

async function submit(input: ProfileInput) {
  submitting.value = true
  error.value = ''
  try {
    const profile = await profiles.create(input)
    await router.push(`/profiles/${profile.id}`)
  } catch (err) {
    error.value = toUserMessage(err, '保存失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.new-profile { max-width: 560px; }
</style>