<template>
  <section>
    <AppPageHeader :title="$t('router.profiles')" :subtitle="$t('profiles.list.subtitle')">
      <template #actions>
        <el-button type="primary" @click="router.push('/profiles/new')">{{ $t('profiles.create') }}</el-button>
      </template>
    </AppPageHeader>

    <el-result
      v-if="error && !profiles.list.length"
      icon="warning"
      :title="$t('profiles.detail.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button type="primary" @click="load">{{ $t('common.retry') }}</el-button>
      </template>
    </el-result>
    <el-skeleton v-else-if="profiles.loading && !profiles.list.length" :rows="5" animated aria-busy="true"/>

    <el-empty
      v-else-if="!profiles.loading && !profiles.list.length"
      :description="$t('profiles.list.empty')"
    >
      <el-button type="primary" tag="a" to="/profiles/new">{{ $t('profiles.create') }}</el-button>
    </el-empty>

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
import { useI18n } from 'vue-i18n'
import { useProfilesStore } from '@/stores/profiles'
import type { Profile } from '@/api/profiles'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import ProfileList from '@/components/profiles/ProfileList.vue'

const router = useRouter()
const { t } = useI18n()
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
    await ElMessageBox.confirm(
      t('profiles.list.deleteConfirm', { nickname: profile.nickname }),
      t('profiles.list.deleteTitle'),
      { type: 'warning', confirmButtonText: t('common.submit'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  try {
    await profiles.remove(profile.id)
  } catch (err) {
    profiles.error = toUserMessage(err, t('profiles.deleteFailed'))
  }
}

onMounted(load)
</script>
