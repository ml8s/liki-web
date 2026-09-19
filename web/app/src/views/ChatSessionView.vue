<template>
  <section class="chat-page">
    <div class="chat-main">
      <header class="session-header">
        <div class="header-nav">
          <router-link v-if="chat.activeSession?.profile_id" :to="`/profiles/${chat.activeSession.profile_id}`">← 返回档案</router-link>
          <router-link v-else to="/chat">← 会话列表</router-link>
        </div>
        <h1>{{ chat.activeSession?.title || '加载中' }}</h1>
        <p v-if="chat.activeSession" class="skill-badge">{{ skillLabel(chat.activeSession.skill) }}</p>
      </header>

      <div
        ref="threadElement"
        v-loading="chat.loadingMessages"
        class="thread"
        @scroll="onThreadScroll"
      >
        <ChatThread :messages="chat.messages" :loading="chat.loadingMessages" />
      </div>

      <ChatStreamStatus :phase="chat.phase" />
      <div v-if="chat.error" class="error-row">
        <AppAlert :title="chat.error" type="error" class="composer-alert" />
        <el-button size="small" @click="retry">重试</el-button>
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
import { useRoute } from 'vue-router'
import { useChatStore, skillLabel } from '@/stores/chat'
import AppAlert from '@/components/app/AppAlert.vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import ChatThread from '@/components/chat/ChatThread.vue'

const route = useRoute()
const chat = useChatStore()
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

function onThreadScroll() {
  const el = threadElement.value
  if (!el) return
  nearBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
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
  document.removeEventListener('keydown', onGlobalKeydown)
  visualViewport?.removeEventListener('resize', onVvResize)
  chat.resetActive()
})
</script>

<style scoped>
.chat-page { min-height: calc(100dvh - 130px); }
.chat-main {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - 130px);
}
.session-header {
  padding-bottom: 12px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--liki-line);
}
.header-nav { margin-bottom: 4px; }
.session-header a {
  display: inline-block;
  padding: 4px 0;
  color: var(--liki-primary);
  text-decoration: none;
  font-size: 13px;
}
.session-header h1 {
  margin: 6px 0 6px;
  font-family: var(--liki-font-serif);
  font-size: 22px;
  line-height: 1.3;
}
.skill-badge {
  display: inline-block;
  margin: 0;
  padding: 2px 10px;
  font-size: 12px;
  color: var(--liki-gold-text);
  border: 1px solid var(--liki-gold);
  border-radius: 4px;
}
.thread { flex: 1; overflow: auto; padding-right: 8px; }
.composer-alert { margin: 12px 0 0; }
.error-row { display: flex; align-items: flex-start; gap: 10px; }
.error-row .el-button { margin-top: 12px; }
</style>
