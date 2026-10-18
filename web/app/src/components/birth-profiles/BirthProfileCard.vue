<template>
  <div class="profile-card">
    <router-link
      class="profile-main"
      :to="`/birth-profiles/${profile.id}`"
    >
      <div class="avatar font-serif">
        {{ initial }}
      </div>
      <div class="info">
        <div class="name-row">
          <span class="name font-serif">{{ profile.nickname }}</span>
        </div>
        <div class="meta">
          {{ birthSummary(profile) }}
        </div>
      </div>
    </router-link>
    <div class="actions">
      <el-button
        size="small"
        link
        type="primary"
        @click="$emit('open', profile)"
      >
        {{ $t('profiles.card.view') }}
      </el-button>
      <el-button
        size="small"
        link
        @click="$emit('edit', profile)"
      >
        {{ $t('profiles.info.edit') }}
      </el-button>
      <el-button
        size="small"
        link
        type="danger"
        class="remove"
        @click="$emit('remove', profile)"
      >
        {{ $t('profiles.card.remove') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { birthSummary, type BirthProfile } from '@/api/birth-profiles'

const props = defineProps<{ profile: BirthProfile }>()
defineEmits<{ open: [BirthProfile]; edit: [BirthProfile]; remove: [BirthProfile] }>()

const initial = computed(() => props.profile.nickname.slice(0, 1))
</script>

<style scoped>
.profile-card {
  display: flex; align-items: center; gap: 14px;
  background: var(--liki-surface); border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg); padding: 16px 18px;
  transition: border-color var(--liki-dur) var(--liki-ease), box-shadow var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease);
}
.profile-main {
  display: flex; align-items: center; gap: 14px; flex: 1; min-width: 0;
  color: inherit; text-decoration: none; cursor: pointer;
}
.profile-card:hover {
  border-color: var(--liki-primary-border);
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-1px);
}
.avatar {
  width: 44px; height: 44px; border-radius: 50%; flex-shrink: 0;
  background: var(--liki-primary-tint); color: var(--liki-primary);
  display: grid; place-items: center; font-size: var(--font-size-title-xs);
}
.info { flex: 1; min-width: 0; }
.name-row { display: flex; align-items: center; gap: 8px; min-width: 0; }
/* 长昵称单行截断，与 Dashboard 的 .rp-name 一致 */
.name {
  overflow: hidden;
  font-size: var(--font-size-title-xs);
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.meta { color: var(--liki-text-2); font-size: var(--font-size-label); margin-top: 3px; }
.actions { display: flex; gap: 4px; }
@media (max-width: 640px) {
  .actions { flex-direction: column; align-items: stretch; gap: 0; }
  .actions :deep(.el-button) { margin: 0; min-height: 40px; padding: var(--space-2) var(--space-3); }
  /* 竖排时删除与其它操作分隔：沿用 RecordActions 的分隔语汇（方向改为上下） */
  .actions .remove {
    margin-top: var(--space-2);
    border-top: 1px solid var(--liki-line);
    border-radius: 0;
  }
}
</style>
