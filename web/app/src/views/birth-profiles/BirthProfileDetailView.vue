<template>
  <section>
    <el-page-header
      class="profile-page-header"
      @back="router.push('/birth-profiles')"
      @keydown="onPageHeaderKeydown"
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
      icon="error"
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
        :error="infoError"
        @update="update"
      />

      <ChartPreview
        :profile="detail"
        auto
      />

      <el-card
        class="cta"
        :title="$t('profiles.detail.interpretation')"
      >
        <!-- 权益未就绪时不得渲染解锁引导：error 提示可重试，未加载完成显示骨架 -->
        <template v-if="entitlements.error">
          <el-alert
            :title="entitlements.error"
            type="error"
            :closable="false"
            show-icon
          />
          <el-button
            type="primary"
            class="cta-retry"
            @click="entitlements.load(true)"
          >
            {{ $t('common.retry') }}
          </el-button>
        </template>

        <el-skeleton
          v-else-if="!entitlements.loaded"
          :rows="2"
          animated
          aria-busy="true"
        />

        <template v-else-if="entitlements.purchased">
          <p class="hint">
            {{ preselectedService
              ? $t('profiles.detail.serviceSelected', { service: $t(preselectedService.labelKey) })
              : $t('profiles.detail.servicePrompt') }}
          </p>
          <IntentEntryList
            :items="intentEntries"
            :selected-id="preselectedKey"
            @select="startService"
          />
        </template>

        <template v-else>
          <p class="hint">
            {{ $t('profiles.detail.unlockHint') }}
          </p>
          <el-button
            type="primary"
            size="large"
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
import { chatUrl, intentById, intentEntryItems } from '@/config/chat'
import ChartPreview from '@/components/birth-profiles/ChartPreview.vue'
import BirthProfileInfo from '@/components/birth-profiles/BirthProfileInfo.vue'
import IntentEntryList from '@/components/app/IntentEntryList.vue'
import { useUnloadGuard } from '@/composables/useUnloadGuard'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const profiles = useBirthProfilesStore()
const entitlements = useEntitlementsStore()
const auth = useAuthStore()

const infoRef = ref<InstanceType<typeof BirthProfileInfo> | null>(null)
const updating = ref(false)
const preselectedKey = ref('')
const drafts = useFormDraftsStore()
const profileId = computed(() => String(route.params.id ?? ''))
const detail = computed(() => profiles.detail)
const error = computed(() => profiles.error)
const preselectedService = computed(() => (preselectedKey.value ? intentById(preselectedKey.value) : undefined))
const intentEntries = computed(() => intentEntryItems(t))
const draftKey = computed(() => auth.user?.id ? `birth-profile:edit:${auth.user.id}:${profileId.value}` : undefined)
const editingInfo = computed(() => Boolean(infoRef.value?.isEditing && infoRef.value?.isDirty))

// 编辑态下的 error 只可能来自保存（加载失败走上面的 el-result）：版本冲突必须给出原因。
const infoError = computed(() => (infoRef.value?.isEditing ? error.value : ''))

useUnloadGuard(editingInfo)

// el-page-header 的返回区是 div[role=button]（EP 只绑了 click）：补齐键盘 Enter/Space 激活。
function onPageHeaderKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  if (!(event.target as HTMLElement | null)?.closest('.el-page-header__back')) return
  event.preventDefault()
  void router.push('/birth-profiles')
}

async function load() {
  await profiles.loadDetail(profileId.value)
}

async function update(input: BirthProfileInput) {
  if (!detail.value) return
  updating.value = true
  profiles.error = ''
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
  drafts.clear('service')
  preselectedKey.value = ''
  void router.push(chatUrl(profileId.value, key))
}

onMounted(async () => {
  const key = drafts.consume<string>('service')
  if (key && intentById(key)) preselectedKey.value = key
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
.cta-retry { margin-top: var(--space-3); }
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
</style>
