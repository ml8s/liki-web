<template>
  <el-alert
    v-if="error"
    :title="error"
    type="error"
    :closable="false"
    show-icon
  />
  <div
    v-else
    class="record-list"
  >
    <div
      v-for="record in records"
      :key="record.id"
      class="record-row"
    >
      <div class="record-main">
        <div class="record-title">
          {{ record.title }}
        </div>
        <div class="record-time">
          {{ formatDateTime(record.updated_at) }}
        </div>
      </div>
      <RecordActions
        :actions="actions"
        @action="key => onAction(key, record)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatDateTime } from '@/utils/format'
import RecordActions, { type RecordAction } from '@/components/app/RecordActions.vue'
import type { ToolRecord } from '@/api/records'

defineProps<{ records: ToolRecord[]; error?: string }>()
const emit = defineEmits<{
  open: [record: ToolRecord]
  rename: [record: ToolRecord]
  remove: [record: ToolRecord]
}>()

const { t } = useI18n()
const actions: RecordAction[] = [
  { key: 'open', label: t('records.open') },
  { key: 'rename', label: t('records.rename') },
  { key: 'remove', label: t('records.delete'), danger: true },
]

function onAction(key: string, record: ToolRecord) {
  if (key === 'open') emit('open', record)
  else if (key === 'rename') emit('rename', record)
  else if (key === 'remove') emit('remove', record)
}
</script>

<style scoped>
.record-list { border: 1px solid var(--liki-line); border-radius: var(--radius-small); background: var(--liki-surface); }
.record-row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 12px; border-bottom: 1px solid var(--liki-line);
}
.record-row:last-child { border-bottom: 0; }
.record-main { min-width: 0; }
.record-title { font-weight: 600; font-size: var(--font-size-label); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.record-time { color: var(--liki-text-3); font-size: var(--font-size-caption); margin-top: 2px; }
</style>
