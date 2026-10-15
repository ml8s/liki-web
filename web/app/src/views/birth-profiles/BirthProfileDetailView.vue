<template>
  <section>
    <el-page-header
      class="profile-page-header"
      @back="router.push('/birth-profiles')"
    >
      <template #content>
        <div class="page-heading">
          <h1>{{ detail?.nickname ?? $t('router.profileDetail') }}</h1>
          <p>{{ $t('profiles.detail.subtitle') }}</p>
        </div>
      </template>
    </el-page-header>

    <el-skeleton
      v-if="profiles.detailLoading"
      :rows="6"
      animated
      aria-busy="true"
    />
    <el-result
      v-else-if="error && !detail"
      icon="warning"
      :title="$t('profiles.detail.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="load"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>

    <template v-else-if="detail">
      <BirthProfileInfo
        ref="infoRef"
        :draft-key="draftKey"
        :profile="detail"
        :submitting="updating"
        @update="update"
      />

      <PaipanPreview
        :profile="detail"
        auto
      />

      <el-card
        class="cta"
        :title="$t('profiles.detail.interpretation')"
      >
        <template v-if="entitlements.loaded && entitlements.purchased">
          <p class="hint">
            {{ preselectedService
              ? $t('profiles.detail.serviceSelected', { service: $t(preselectedService.labelKey) })
              : $t('profiles.detail.servicePrompt') }}
          </p>
          <div class="service-chips">
            <el-button
              v-for="s in chatServices"
              :key="s.key"
              :type="s.key === preselectedKey ? 'primary' : 'default'"
              @click="startService(s.key)"
            >
              {{ $t(s.labelKey) }}
            </el-button>
          </div>
        </template>

        <template v-else>
          <p class="hint">
            {{ $t('profiles.detail.unlockHint') }}
          </p>
          <el-button
            type="primary"
            size="large"
            :loading="entitlements.loading"
            @click="goChat"
          >
            {{ $t('profiles.detail.unlock') }}
          </el-button>
        </template>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useAuthStore } from '@/stores/auth'
import type { BirthProfileInput } from '@/api/birth-profiles'
import { chatServices, chatUrl, serviceByKey } from '@/config/chat'
import PaipanPreview from '@/components/birth-profiles/PaipanPreview.vue'
import BirthProfileInfo from '@/components/birth-profiles/BirthProfileInfo.vue'
import { useUnloadGuard } from '@/composables/useUnloadGuard'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const profiles = useBirthProfilesStore()
const entitlements = useEntitlementsStore()
const auth = useAuthStore()

const infoRef = ref<InstanceType<typeof BirthProfileInfo> | null>(null)
const updating = ref(false)
const preselectedKey = ref('')
const profileId = computed(() => String(route.params.id ?? ''))
const detail = computed(() => profiles.detail)
const error = computed(() => profiles.error)
const preselectedService = computed(() => (preselectedKey.value ? serviceByKey(preselectedKey.value) : undefined))
const draftKey = computed(() => auth.user?.id ? `birth-profile:edit:${auth.user.id}:${profileId.value}` : undefined)
const editingInfo = computed(() => Boolean(infoRef.value?.isEditing && infoRef.value?.isDirty))

useUnloadGuard(editingInfo)

async function load() {
  await profiles.loadDetail(profileId.value)
}

async function update(input: BirthProfileInput) {
  if (!detail.value) return
  updating.value = true
  try {
    await profiles.update(detail.value.id, detail.value.version, input)
    infoRef.value?.closeEdit()
  } catch (err) {
    profiles.error = toUserMessage(err, t('profiles.saveFailed'))
  } finally {
    updating.value = false
  }
}

async function goChat() {
  await entitlements.load()
  void router.push({ path: '/chat', query: entitlements.purchased ? { profile: profileId.value } : undefined })
}

function startService(key: string) {
  try { sessionStorage.removeItem('liki-service') } catch { /* ignore */ }
  preselectedKey.value = ''
  void router.push(chatUrl(profileId.value, key))
}

onMounted(async () => {
  try {
    const key = sessionStorage.getItem('liki-service')
    if (key && serviceByKey(key)) preselectedKey.value = key
  } catch { /* ignore */ }
  try {
    await Promise.all([load(), entitlements.load()])
  } catch (error) {
    console.error('[BirthProfileDetail] 初始化失败:', error)
  }
})

onBeforeRouteLeave(async () => {
  if (!infoRef.value?.isEditing || !infoRef.value?.isDirty) return true
  try {
    await ElMessageBox.confirm(
      t('profiles.detail.unsavedBody'),
      t('profiles.detail.unsavedTitle'),
      { type: 'warning', confirmButtonText: t('profiles.detail.leave'), cancelButtonText: t('profiles.detail.keepEditing') },
    )
    infoRef.value?.clearDraft()
    return true
  } catch {
    return false
  }
})
</script>

<style scoped>
.cta { margin-top: 8px; }
.profile-page-header { margin-bottom: 20px; }
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
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
.service-chips { display: flex; flex-wrap: wrap; gap: 10px; }
</style>
