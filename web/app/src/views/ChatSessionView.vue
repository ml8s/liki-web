<template>
  <section class="chat-page">
    <div class="chat-main">
      <el-page-header
        class="session-header"
        @back="goBack"
      >
        <template #content>
          <div class="page-heading">
            <h1>{{ chat.activeSession?.title || $t('common.loading') }}</h1>
            <el-tag
              v-if="chat.activeSession"
              size="small"
            >
              {{ skillLabel(chat.activeSession.expert) }}
            </el-tag>
          </div>
        </template>
      </el-page-header>

      <div
        ref="threadElement"
        v-loading="chat.loadingMessages"
        :aria-busy="chat.loadingMessages"
        class="thread"
        @scroll="onThreadScroll"
      >
        <ChatThread
          :messages="chat.messages"
          :loading="chat.loadingMessages"
          @select="draft = $event"
        />
      </div>

      <ChatStreamStatus :phase="chat.phase" />
      <div
        v-if="chat.error"
        class="error-row"
      >
        <el-alert
          :title="chat.error"
          type="error"
          class="composer-alert"
          :closable="false"
          show-icon
        />
        <el-button
          size="small"
          @click="retry"
        >
          {{ $t('common.retry') }}
        </el-button>
      </div>

      <ChatComposer
        v-model="draft"
        :sending="chat.sending"
        @submit="submit"
        @stop="chat.stopGeneration()"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import ChatThread from '@/components/chat/ChatThread.vue'

const route = useRoute()
const chat = useChatStore()
const router = useRouter()
const draft = ref('')
const threadElement = ref<HTMLElement | null>(null)
const nearBottom = ref(true)
const sessionId = computed(() => String(route.params.sessionId ?? ''))
const DRAFT_KEY = 'likiChatDraft'

// 输入框草稿持久化：刷新/离开页面后恢复（借命名 chat 实践）。
watch(draft, value => {
  try {
    if (value) sessionStorage.setItem(DRAFT_KEY, value)
    else sessionStorage.removeItem(DRAFT_KEY)
  } catch {
    // ignore
  }
})

function restoreDraft() {
  try {
    const saved = sessionStorage.getItem(DRAFT_KEY)
    if (saved) draft.value = saved
  } catch {
    // ignore
  }
}

function goBack() {
  const profileId = chat.activeSession?.profile_id
  void router.push(profileId ? `/birth-profiles/${profileId}` : '/chat')
}

let threadScrollFrame: number | null = null

function measureThreadScroll() {
  const el = threadElement.value
  if (!el) return
  nearBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

function onThreadScroll() {
  if (threadScrollFrame !== null) return
  threadScrollFrame = requestAnimationFrame(() => {
    threadScrollFrame = null
    measureThreadScroll()
  })
}

async function scrollToEnd() {
  await nextTick()
  const el = threadElement.value
  if (el && nearBottom.value) {
    el.scrollTop = el.scrollHeight
  }
}

async function submit() {
  const message = draft.value.trim()
  if (!message) return
  draft.value = ''
  await chat.send(message)
  if (chat.error) draft.value = message
  nearBottom.value = true
  await scrollToEnd()
}

function retry() {
  const message = draft.value.trim()
  if (!message || chat.sending) return
  draft.value = ''
  void chat.send(message)
}

watch(sessionId, async id => {
  if (id) {
    try {
      await chat.openSession(id)
      // 服务入口：会话打开后自动发出首条意图（内存态，刷新即失不重发）。
      if (chat.pendingIntent) {
        const intent = chat.pendingIntent
        chat.pendingIntent = ''
        await chat.send(intent)
      }
      await scrollToEnd()
    } catch (error) {
      // store 内部已设 error state；此处防御编程错误导致的 unhandled rejection。
      console.error('[ChatSession] 会话加载/发送失败:', error)
    }
  }
}, { immediate: true })

watch(() => chat.messages.length, () => scrollToEnd())

function onGlobalKeydown(event: KeyboardEvent) {
  // Esc 停止生成（借命名 chat 实践）。
  if (event.key === 'Escape' && chat.sending) {
    chat.stopGeneration()
  }
}

let visualViewport: VisualViewport | null = null

// 移动端键盘开合（visualViewport 高度变化）时跟随到底，保证输入框可见（借命名 chat 实践）。
function onVvResize() {
  nearBottom.value = true
  void scrollToEnd()
}

onMounted(() => {
  restoreDraft()
  document.addEventListener('keydown', onGlobalKeydown)
  if (typeof window !== 'undefined' && window.visualViewport) {
    visualViewport = window.visualViewport
    visualViewport.addEventListener('resize', onVvResize)
  }
  void scrollToEnd()
})

onBeforeUnmount(() => {
  if (threadScrollFrame !== null) cancelAnimationFrame(threadScrollFrame)
  document.removeEventListener('keydown', onGlobalKeydown)
  visualViewport?.removeEventListener('resize', onVvResize)
  chat.resetActive()
})
</script>

<style scoped>
.chat-page {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
.chat-main {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}
.session-header {
  margin-bottom: 16px;
}
.page-heading {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.page-heading h1 {
  margin: 0;
  overflow: hidden;
  font-size: var(--font-size-title-md);
  font-weight: 650;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.thread { flex: 1; overflow: auto; padding-right: 8px; }
.composer-alert { margin: 12px 0 0; }
.error-row { display: flex; align-items: flex-start; gap: 10px; }
.error-row .el-button { margin-top: 12px; }
</style>
