<template>
  <!-- 桌面：表格 -->
  <el-table
    v-if="!isMobile"
    :data="sessions"
    @row-click="$emit('open', $event)"
  >
    <el-table-column
      prop="title"
      :label="$t('chat.sessions.title')"
      min-width="180"
    />
    <el-table-column
      :label="$t('chat.sessions.updated')"
      width="190"
    >
      <template #default="{ row }">
        {{ formatDateTime(row.updated_at) }}
      </template>
    </el-table-column>
    <el-table-column
      width="180"
      align="right"
    >
      <template #default="{ row }">
        <RecordActions
          :actions="desktopActions(asSession(row))"
          @action="key => emitAction(key, asSession(row))"
        />
      </template>
    </el-table-column>
  </el-table>

  <!-- 移动端：卡片降级 -->
  <div
    v-else
    class="session-cards"
  >
    <div
      v-for="s in sessions"
      :key="s.id"
      class="session-card"
    >
      <router-link
        :to="`/chat/${s.id}`"
        class="session-card-main"
      >
        <div class="sc-top">
          <span class="sc-title">{{ s.title }}</span>
        </div>
        <div class="sc-time">
          {{ formatDateTime(s.updated_at) }}
        </div>
      </router-link>
      <RecordActions
        class="sc-actions"
        :actions="mobileActions()"
        @action="key => emitAction(key, s)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ChatSession } from '@/api/chat'
import { formatDateTime } from '@/utils/format'
import { useMobile } from '@/composables/useMobile'
import { useI18n } from 'vue-i18n'
import RecordActions, { type RecordAction } from '@/components/app/RecordActions.vue'

defineProps<{
  sessions: ChatSession[]
}>()

const emit = defineEmits<{ open: [session: ChatSession]; rename: [session: ChatSession]; remove: [session: ChatSession] }>()

const { t } = useI18n()
const isMobile = useMobile(640)

function asSession(row: Record<string, unknown>): ChatSession {
  return row as unknown as ChatSession
}

function desktopActions(session: ChatSession): RecordAction[] {
  return [
    { key: 'open', label: t('chat.sessions.open'), to: `/chat/${session.id}` },
    ...mobileActions(),
  ]
}

function mobileActions(): RecordAction[] {
  return [
    { key: 'rename', label: t('chat.sessions.rename') },
    { key: 'remove', label: t('chat.sessions.delete'), danger: true },
  ]
}

function emitAction(key: string, session: ChatSession) {
  if (key === 'rename') emit('rename', session)
  else if (key === 'remove') emit('remove', session)
  else if (key === 'open') emit('open', session)
}
</script>

<style scoped>
.session-cards { display: grid; gap: 12px; }
.session-card {
  background: var(--liki-surface);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  padding: 14px 16px;
  transition: border-color var(--liki-dur) var(--liki-ease), transform var(--liki-dur) var(--liki-ease), box-shadow var(--liki-dur) var(--liki-ease);
}
.session-card:hover {
  border-color: var(--liki-primary-border);
  transform: translateY(-1px);
  box-shadow: var(--shadow-card-hover);
}
.session-card-main { display: block; text-decoration: none; color: inherit; }
.sc-actions { margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--liki-line); justify-content: flex-start; }
.sc-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.sc-title { font-weight: 600; }
.sc-time { color: var(--liki-text-2); font-size: var(--font-size-label); margin-top: 6px; }
</style>
