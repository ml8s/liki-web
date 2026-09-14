<template>
  <section>
    <AppPageHeader :title="detail?.nickname ?? '档案详情'" subtitle="查看信息、免费预览排盘或开启解读。" />

    <AppSkeleton v-if="profiles.detailLoading" :rows="6" />
    <AppErrorState v-else-if="error && !detail" title="档案加载失败" :description="error" @retry="load" />

    <template v-else-if="detail">
      <ProfileInfo :profile="detail" :submitting="updating" @update="update" />

      <PaipanPreview :profile="detail" />

      <el-card class="cta" shadow="never">
        <template #header><span class="card-title">解读</span></template>
        <p class="hint">{{ entitlements.loaded && purchased ? '可围绕此人的出生信息自由对话。' : '解锁 Liki Chat 后可围绕此人的出生信息自由对话。' }}</p>
        <el-button type="primary" size="large" :loading="entitlements.loading" @click="goChat">
          {{ purchased ? '开始解读' : '解锁 Chat' }}
        </el-button>
      </el-card>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProfilesStore } from '@/stores/profiles'
import { useEntitlementsStore } from '@/stores/entitlements'
import type { ProfileInput } from '@/api/profiles'
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
const profileId = computed(() => String(route.params.id ?? ''))
const detail = computed(() => profiles.detail)
const error = computed(() => profiles.error)
const purchased = computed(() => entitlements.allowedSkills.length === 4)

async function load() {
  await profiles.loadDetail(profileId.value)
}

async function update(input: ProfileInput) {
  if (!detail.value) return
  updating.value = true
  try {
    await profiles.update(detail.value.id, detail.value.version, input)
  } catch (err) {
    profiles.error = err instanceof Error ? err.message : '保存失败'
  } finally {
    updating.value = false
  }
}

async function goChat() {
  await entitlements.load()
  void router.push({ path: '/chat', query: purchased.value ? { profile: profileId.value } : undefined })
}

onMounted(async () => {
  await Promise.all([load(), entitlements.load()])
})
</script>

<style scoped>
.cta { border-radius: 12px; margin-top: 8px; }
.card-title { font-weight: 600; }
.hint { color: var(--liki-text-2); margin: 0 0 14px; }
</style>