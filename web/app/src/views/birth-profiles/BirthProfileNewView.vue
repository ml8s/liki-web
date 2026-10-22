<template>
  <section class="new-profile">
    <el-page-header
      class="new-profile-header"
      @back="router.push('/birth-profiles')"
      @keydown="onPageHeaderKeydown"
    >
      <template #content>
        <div class="page-heading">
          <h1>{{ $t('router.profileNew') }}</h1>
          <p>{{ $t('profiles.new.subtitle') }}</p>
        </div>
      </template>
    </el-page-header>
    <div
      v-if="draftLoaded || error"
      class="alerts"
    >
      <el-alert
        v-if="draftLoaded"
        :title="$t('profiles.new.draftLoaded')"
        type="info"
        :closable="false"
        show-icon
      />
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        :closable="false"
        show-icon
      />
    </div>
    <BirthForm
      ref="birthFormRef"
      :submitting="submitting"
      :initial="draft ?? undefined"
      :draft-key="draftKey"
      @dirty="isDirty = $event"
      @submit="submit"
    />
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, ref } from 'vue'
import { onBeforeRouteLeave, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useAuthStore } from '@/stores/auth'
import { useUnloadGuard } from '@/composables/useUnloadGuard'
import type { BirthProfileInput } from '@/api/birth-profiles'
import BirthForm from '@/components/birth-profiles/BirthForm.vue'
import { consumeHandoff, HANDOFF_KEYS } from '@shared/utils/handoff'
import type { BirthFieldsValue } from '@shared/components/form/birth-fields'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const profiles = useBirthProfilesStore()
const submitting = ref(false)
const error = ref('')
const draft = ref<Partial<BirthProfileInput> | null>(null)
const draftLoaded = ref(false)
const isDirty = ref(false)
const birthFormRef = ref<InstanceType<typeof BirthForm> | null>(null)
const draftKey = computed(() => auth.user?.id ? `birth-profile:new:${auth.user.id}` : undefined)

useUnloadGuard(isDirty)

// el-page-header 的返回区是 div[role=button]（EP 只绑了 click）：补齐键盘 Enter/Space 激活。
function onPageHeaderKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  if (!(event.target as HTMLElement | null)?.closest('.el-page-header__back')) return
  event.preventDefault()
  void router.push('/birth-profiles')
}

onBeforeRouteLeave(async () => {
  if (!isDirty.value) return true
  try {
    await ElMessageBox.confirm(
      t('profiles.detail.unsavedBody'),
      t('profiles.detail.unsavedTitle'),
      {
        type: 'warning',
        confirmButtonText: t('profiles.detail.leave'),
        cancelButtonText: t('profiles.detail.keepEditing'),
      },
      )
      birthFormRef.value?.clearDraft()
      return true
  } catch {
    return false
  }
})

// 跨端 handoff（营销站→App）：读后即清，带 TTL。
const handoff = consumeHandoff<BirthFieldsValue>(HANDOFF_KEYS.birth)
if (handoff?.date && handoff.location) {
  draft.value = {
    birth: {
      date: handoff.date,
      time: handoff.time,
      gender: handoff.gender === 'male' ? 'male' : 'female',
      location: {
        place: handoff.location.place,
        longitude: handoff.location.longitude,
        timezone: handoff.location.timezone,
      },
    },
    nickname: '',
    third_party_consent: false,
  }
  draftLoaded.value = true
}

async function submit(input: BirthProfileInput) {
  submitting.value = true
  error.value = ''
  try {
    const profile = await profiles.create(input)
    isDirty.value = false
    birthFormRef.value?.clearDraft()
    await router.push(`/birth-profiles/${profile.id}`)
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
.alerts { display: grid; gap: var(--space-3); margin-bottom: var(--space-4); }
.page-heading h1 {
  margin: 0;
  font-size: var(--font-size-title-lg);
  font-weight: 650;
  letter-spacing: -0.03em;
}
.page-heading p {
  margin: 4px 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-body);
}
</style>
