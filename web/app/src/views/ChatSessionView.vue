<template>
  <section class="chat-page">
    <div class="chat-main">
      <el-page-header
        class="session-header"
        @back="goBack"
      >
        <template #content>
          <div class="page-heading">
            <h1>{{ chat.activeSession?.title || (sessionUnavailable ? $t('chat.sessionUnavailable') : $t('common.loading')) }}</h1>
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
        <div
          v-if="chat.messagesCursor"
          class="earlier-row"
        >
          <el-button
            size="small"
            :loading="chat.loadingEarlier"
            @click="loadEarlier"
          >
            {{ $t('chat.loadEarlier') }}
          </el-button>
        </div>
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
          v-if="sessionUnavailable"
          size="small"
          @click="goBack"
        >
          {{ $t('chat.backToSessions') }}
        </el-button>
        <el-button
          v-else
          size="small"
          @click="retry"
        >
          {{ $t('common.retry') }}
        </el-button>
      </div>

      <div
        v-if="!sessionUnavailable"
        class="composer-area"
      >
        <button
          v-if="!nearBottom"
          type="button"
          class="jump-latest"
          @click="jumpToLatest"
        >
          {{ $t('chat.jumpToLatest') }}
        </button>
        <ChatComposer
          v-model="draft"
          :sending="chat.sending"
          @submit="submit"
          @stop="chat.stopGeneration()"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatStreamStatus from '@/components/chat/ChatStreamStatus.vue'
import ChatThread from '@/components/chat/ChatThread.vue'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

const route = useRoute()
const chat = useChatStore()
const router = useRouter()
const draft = ref('')
const threadElement = ref<HTMLElement | null>(null)
const nearBottom = ref(true)
const sessionId = computed(() => String(route.params.sessionId ?? ''))
const sessionUnavailable = computed(() =>
  !chat.loadingMessages && !chat.activeSession && !!chat.error)
const DRAFT_KEY = 'chat-draft'
const drafts = useFormDraftsStore()

// 输入框草稿持久化：刷新/离开页面后恢复（借命名 chat 实践）。
watch(draft, value => {
  if (value) drafts.set(DRAFT_KEY, value)
  else drafts.clear(DRAFT_KEY)
})

function restoreDraft() {
  const saved = drafts.get<string>(DRAFT_KEY)
  if (saved) draft.value = saved
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

// 错误有两个来源：发送失败（草稿已恢复，重发草稿）与会话加载失败（草稿为空，重载会话）。
function retry() {
  if (chat.sending) return
  const message = draft.value.trim()
  if (message) {
    draft.value = ''
    void chat.send(message)
    return
  }
  void reloadSession()
}

// 加载更早的消息：顶部插入后把滚动位置下移插入的高度差，视口内容不跳动。
async function loadEarlier() {
  const el = threadElement.value
  const beforeHeight = el?.scrollHeight ?? 0
  const loaded = await chat.loadEarlierMessages()
  if (!loaded || !el) return
  await nextTick()
  el.scrollTop += el.scrollHeight - beforeHeight
}

async function reloadSession() {
  if (!sessionId.value) return
  await chat.openSession(sessionId.value)
  nearBottom.value = true
  await scrollToEnd()
}

function jumpToLatest() {
  nearBottom.value = true
  void scrollToEnd()
}

watch(sessionId, async id => {
  if (id) {
    nearBottom.value = true
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

// 流式期间消息条数不变、只有最后一条的内容在增长：跟踪长度变化才能跟随滚动。
watch(
  () => {
    const last = chat.messages[chat.messages.length - 1]
    return last ? `${chat.messages.length}:${last.content?.length ?? 0}` : ''
  },
  () => scrollToEnd(),
)

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
.thread { flex: 1; overflow: auto; padding-right: 8px; scrollbar-gutter: stable; }
.earlier-row { display: flex; justify-content: center; padding-bottom: var(--space-3); }
.composer-alert { margin: 12px 0 0; }
.error-row { display: flex; align-items: flex-start; gap: 10px; }
.error-row .el-button { margin-top: 12px; }
.composer-area { position: relative; }
.jump-latest {
  position: absolute;
  top: -46px;
  left: 50%;
  transform: translateX(-50%);
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 6px 14px;
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-full);
  background: var(--liki-surface);
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
  cursor: pointer;
  box-shadow: var(--shadow-md);
  white-space: nowrap;
}
.jump-latest:hover { color: var(--liki-primary); border-color: var(--liki-primary-border); }

@media (max-width: 768px) {
  .jump-latest {
    top: -56px;
    min-height: 44px;
    padding: 10px 16px;
  }
}
</style>
