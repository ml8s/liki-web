<template>
  <section class="new-profile">
    <el-page-header class="new-profile-header" @back="router.push('/profiles')">
      <template #content>
        <div class="page-heading">
          <h1>{{ $t('router.profileNew') }}</h1>
          <p>{{ $t('profiles.new.subtitle') }}</p>
        </div>
      </template>
    </el-page-header>
    <el-alert v-if="draftLoaded" :title="$t('profiles.new.draftLoaded')" type="info" :closable="false" show-icon/>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon/>
    <BirthForm :submitting="submitting" :initial="draft ?? undefined" @submit="submit" />
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useProfilesStore } from '@/stores/profiles'
import type { ProfileInput } from '@/api/profiles'
import BirthForm from '@/components/profiles/BirthForm.vue'

const router = useRouter()
const { t } = useI18n()
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
        ...(parsed.gender === 'male' || parsed.gender === 'female' ? { gender: parsed.gender } : {}),
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
    error.value = toUserMessage(err, t('profiles.saveFailed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.new-profile { max-width: 640px; }
.new-profile-header { margin-bottom: 20px; }
.page-heading h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 650;
  letter-spacing: -0.03em;
}
.page-heading p {
  margin: 4px 0 0;
  color: var(--liki-text-2);
  font-size: 14px;
}
</style>
