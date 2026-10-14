<template>
  <el-card
    class="skill-card"
    shadow="never"
  >
    <template #header>
      <div class="app-card__header">
        <h2 class="app-card__title">
          {{ label }}
        </h2>
        <el-tag
          :type="enabled ? 'success' : 'info'"
          size="small"
        >
          {{ enabled ? $t('chat.status.available') : (statusLabel || $t('chat.status.inactive')) }}
        </el-tag>
      </div>
    </template>

    <p>{{ description }}</p>

    <p
      v-if="blockedReason"
      class="blocked-reason"
    >
      {{ blockedReason }}
    </p>

    <el-button
      v-if="!enabled && actionTo"
      tag="a"
      :to="actionTo"
      type="primary"
      plain
    >
      {{ actionLabel || $t('chat.action.resolve') }}
    </el-button>
    <el-button
      type="primary"
      :disabled="!enabled || starting"
      :loading="starting"
      @click="$emit('start')"
    >
      {{ actionLabel || $t('chat.action.start') }}
    </el-button>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  description: string
  enabled: boolean
  starting?: boolean
  blockedReason?: string
  statusLabel?: string
  actionLabel?: string
  actionTo?: string
}>()

defineEmits<{ start: [] }>()
</script>

<style scoped>
p { min-height: 44px; color: var(--liki-text-2); }
.blocked-reason {
  min-height: 0;
  margin: -6px 0 12px;
  color: var(--liki-text-3);
  font-size: 13px;
}
</style>
