<template>
  <section>
    <AppPageHeader :title="detail?.nickname ?? '档案详情'" subtitle="查看信息、免费预览排盘或开启解读。" />

    <AppSkeleton v-if="profiles.detailLoading" :rows="6" />
    <AppErrorState v-else-if="error && !detail" title="档案加载失败" :description="error" @retry="load" />

    <template v-else-if="detail">
      <ProfileInfo :profile="detail" :submitting="updating" @update="update" />

      <PaipanPreview :profile="detail" auto />

      <el-card class="cta" shadow="never">
        <template #header><span class="card-title">解读</span></template>

        <template v-if="entitlements.loaded && purchased">
          <p class="hint">
            {{ preselectedService ? `已为你选择「${preselectedService.label}」，也可以换一项：` : '想了解什么？选择一项进入专业解读。' }}
          </p>
          <div class="service-chips">
            <el-button
              v-for="s in chatServices"
              :key="s.key"
              :type="s.key === preselectedKey ? 'primary' : 'default'"
              @click="startService(s.key)"
            >{{ s.label }}</el-button>
          </div>
        </template>

        <template v-else>
          <p class="hint">解锁 Liki Chat 后可围绕此人的出生信息进行专业解读。</p>
          <el-button type="primary" size="large" :loading="entitlements.loading" @click="goChat">解锁 Chat</el-button>
        </template>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProfilesStore } from '@/stores/profiles'
import { useEntitlementsStore } from '@/stores/entitlements'
import type { ProfileInput } from '@/api/profiles'
import { chatServices, serviceByKey } from '@/config/chat'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'
import PaipanPreview from '@/components/profiles/PaipanPreview.vue'
import ProfileInfo from '@/components/profiles/ProfileInfo.vue'

const route = useRoute()
const router = useRouter()
const profiles = useProfilesStore()
const entitlements = useEntitlementsStore()

const updating = ref(false)
const preselectedKey = ref('')
const profileId = computed(() => String(route.params.id ?? ''))
const detail = computed(() => profiles.detail)
const error = computed(() => profiles.error)
const purchased = computed(() => entitlements.allowedSkills.length > 0)
const preselectedService = computed(() => (preselectedKey.value ? serviceByKey(preselectedKey.value) : undefined))

async function load() {
  await profiles.loadDetail(profileId.value)
}

async function update(input: ProfileInput) {
  if (!detail.value) return
  updating.value = true
  try {
    await profiles.update(detail.value.id, detail.value.version, input)
  } catch (err) {
    profiles.error = toUserMessage(err, '保存失败')
  } finally {
    updating.value = false
  }
}

async function goChat() {
  await entitlements.load()
  void router.push({ path: '/chat', query: purchased.value ? { profile: profileId.value } : undefined })
}

function startService(key: string) {
  try { sessionStorage.removeItem('liki-service') } catch { /* ignore */ }
  preselectedKey.value = ''
  void router.push(`/chat?profile=${profileId.value}&service=${key}`)
}

onMounted(async () => {
  try {
    const key = sessionStorage.getItem('liki-service')
    if (key && serviceByKey(key)) preselectedKey.value = key
  } catch { /* ignore */ }
  await Promise.all([load(), entitlements.load()])
})
</script>

<style scoped>
.cta { border-radius: 12px; margin-top: 8px; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
.service-chips { display: flex; flex-wrap: wrap; gap: 10px; }
</style>