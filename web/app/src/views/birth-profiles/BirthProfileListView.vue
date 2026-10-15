<template>
  <section>
    <AppPageHeader
      :title="$t('router.profiles')"
      :subtitle="$t('profiles.list.subtitle')"
    >
      <template #actions>
        <el-button
          type="primary"
          @click="router.push('/birth-profiles/new')"
        >
          {{ $t('profiles.create') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-result
      v-if="error && !profiles.list.length"
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
    <el-skeleton
      v-else-if="profiles.loading && !profiles.list.length"
      :rows="5"
      animated
      aria-busy="true"
    />

    <el-empty
      v-else-if="!profiles.loading && !profiles.list.length"
      :description="$t('profiles.list.empty')"
    >
      <el-button
        type="primary"
        tag="router-link"
        to="/birth-profiles/new"
      >
        {{ $t('profiles.create') }}
      </el-button>
    </el-empty>

    <BirthProfileList
      v-else
      :profiles="profiles.list"
      @open="open"
      @edit="open"
      @remove="confirmRemove"
    />

    <div
      v-if="profiles.next_cursor"
      class="load-more"
    >
      <el-button
        :loading="profiles.loading"
        @click="profiles.loadMore()"
      >
        {{ $t('profiles.list.loadMore') }}
      </el-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import type { BirthProfile } from '@/api/birth-profiles'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import BirthProfileList from '@/components/birth-profiles/BirthProfileList.vue'

const router = useRouter()
const { t } = useI18n()
const profiles = useBirthProfilesStore()

const error = computed(() => profiles.error)

async function load() {
  await profiles.loadList()
}

function open(profile: BirthProfile) {
  void router.push(`/birth-profiles/${profile.id}`)
}

async function confirmRemove(profile: BirthProfile) {
  try {
    await ElMessageBox.confirm(
      t('profiles.list.deleteConfirm', { nickname: profile.nickname }),
      t('profiles.list.deleteTitle'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') },
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

onMounted(async () => {
  await load()
  if (!profiles.error && !profiles.list.length) {
    void router.replace('/birth-profiles/new')
  }
})
</script>

<style scoped>
.load-more {
  margin-top: 12px;
  text-align: center;
}
</style>
