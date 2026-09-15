<template>
  <section class="chat-page">
    <div class="chat-main">
      <header class="session-header">
        <router-link to="/chat">← 会话列表</router-link>
        <h1>{{ chat.activeSession?.title || '加载中' }}</h1>
        <p v-if="chat.activeSession">领域：{{ skillLabel(chat.activeSession.skill) }}</p>
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
        <AppAlert
          :title="chat.error"
          type="error"
          class="composer-alert"
        />
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
    await chat.openSession(id)
    await scrollToEnd()
  }
}, { immediate: true })

watch(() => chat.messages, () => scrollToEnd(), { deep: true })

onMounted(scrollToEnd)
onBeforeUnmount(() => chat.resetActive())
</script>

<style scoped>
.chat-page { min-height: calc(100dvh - 130px); }
.chat-main {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - 130px);
}
.session-header {
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid #e5e7eb;
}
.session-header h1 { margin: 6px 0; font-size: 22px; }
.session-header p { margin: 0; color: #666; }
.thread { flex: 1; overflow: auto; padding-right: 8px; }
.composer-alert { margin: 12px 0 0; }
.error-row { display: flex; align-items: flex-start; gap: 10px; }
.error-row .el-button { margin-top: 14px; }
</style>
