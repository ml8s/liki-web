<template>
  <el-card class="app-card profile-info" shadow="never">
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">{{ $t('profiles.info.title') }}</h2>
        <div class="app-card__actions">
          <el-button v-if="!editing" size="small" text type="primary" @click="editing = true">{{ $t('profiles.info.edit') }}</el-button>
          <el-button v-else size="small" text @click="cancelEdit">{{ $t('common.cancel') }}</el-button>
        </div>
      </div>
    </template>

    <BirthForm
      v-if="editing"
      :submitting="submitting"
      :initial="profile"
      @dirty="dirty = $event"
      @submit="$emit('update', $event)"
    />

    <el-descriptions v-else :column="1" border>
      <el-descriptions-item :label="$t('profiles.form.nickname')">{{ profile.nickname }}</el-descriptions-item>
      <el-descriptions-item v-if="profile.name" :label="$t('profiles.info.name')">{{ profile.name }}</el-descriptions-item>
      <el-descriptions-item :label="$t('profiles.form.gender')">{{ $t(profile.gender === 'male' ? 'profiles.male' : 'profiles.female') }}</el-descriptions-item>
      <el-descriptions-item :label="$t('profiles.info.birth')">{{ birthSummary(profile) }}</el-descriptions-item>
      <el-descriptions-item v-if="profile.note" :label="$t('profiles.form.note')">{{ profile.note }}</el-descriptions-item>
      <el-descriptions-item :label="$t('profiles.info.version')">v{{ profile.version }}</el-descriptions-item>
    </el-descriptions>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { birthSummary, type Profile, type ProfileInput } from '@/api/profiles'
import BirthForm from '@/components/profiles/BirthForm.vue'
defineProps<{ profile: Profile; submitting?: boolean }>()
defineEmits<{ update: [ProfileInput] }>()
const { t } = useI18n()

const editing = ref(false)
const dirty = ref(false)

async function cancelEdit() {
  if (!dirty.value) {
    editing.value = false
    return
  }
  try {
    await ElMessageBox.confirm(
      t('profiles.info.unsavedBody'),
      t('profiles.detail.unsavedTitle'),
      { type: 'warning', confirmButtonText: t('profiles.detail.leave'), cancelButtonText: t('profiles.detail.keepEditing') },
    )
    editing.value = false
    dirty.value = false
  } catch {
    // 用户选择继续编辑。
  }
}

defineExpose({
  isEditing: editing,
  isDirty: dirty,
  closeEdit() {
    editing.value = false
    dirty.value = false
  },
})
</script>

<style scoped>
.profile-info { margin-bottom: 20px; }
</style>
