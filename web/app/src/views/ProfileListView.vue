<template>
  <section>
    <AppPageHeader title="档案" subtitle="管理你关心的人的出生信息。">
      <template #actions>
        <el-button type="primary" @click="router.push('/profiles/new')">新建档案</el-button>
      </template>
    </AppPageHeader>

    <AppErrorState v-if="error && !profiles.list.length" title="档案加载失败" :description="error" @retry="load" />
    <AppSkeleton v-if="profiles.loading && !profiles.list.length" :rows="5" />

    <AppEmpty v-else-if="!profiles.loading && !profiles.list.length" description="还没有档案，先创建一个吧" />

    <ProfileList
      v-else
      :profiles="profiles.list"
      @open="open"
      @edit="open"
      @remove="confirmRemove"
    />
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProfilesStore } from '@/stores/profiles'
import type { Profile } from '@/api/profiles'
import AppEmpty from '@/components/app/AppEmpty.vue'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'
import ProfileList from '@/components/profiles/ProfileList.vue'

const router = useRouter()
const profiles = useProfilesStore()

const error = profiles.error

async function load() {
  await profiles.loadList()
}

function open(profile: Profile) {
  void router.push(`/profiles/${profile.id}`)
}

async function confirmRemove(profile: Profile) {
  try {
    await ElMessageBox.confirm(`删除档案「${profile.nickname}」？30 天内可恢复。`, '删除档案', { type: 'warning' })
  } catch {
    return
  }
  try {
    await profiles.remove(profile.id)
  } catch (err) {
    profiles.error = toUserMessage(err, '删除失败')
  }
}

onMounted(load)
</script>