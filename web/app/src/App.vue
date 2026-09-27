<template>
  <el-provider :locale="elementLocale">
    <div
      v-if="fatal.crashed"
      class="fatal"
      role="alert"
    >
      <h1>{{ $t('fatal.title') }}</h1>
      <p class="msg">
        {{ fatal.message || $t('fatal.defaultMessage') }}
      </p>
      <div class="actions">
        <el-button @click="retry">
          {{ $t('fatal.backHome') }}
        </el-button>
        <el-button
          type="primary"
          @click="reload"
        >
          {{ $t('fatal.reload') }}
        </el-button>
      </div>
    </div>
    <router-view
      v-else
      v-slot="{ Component }"
    >
      <transition name="page-fade">
        <component :is="Component" />
      </transition>
    </router-view>
  </el-provider>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ElConfigProvider as ElProvider } from 'element-plus'
import en from 'element-plus/es/locale/lang/en'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { fatal } from '@/errors'

const router = useRouter()
const { locale } = useI18n()
const elementLocale = computed(() => {
  if (locale.value === 'zh-Hans') return zhCn
  if (locale.value === 'en') return en
  return zhTw
})

function reload() {
  window.location.reload()
}

function retry() {
  fatal.crashed = false
  fatal.message = ''
  void router.push('/')
}
</script>

<style scoped>
.fatal {
  min-height: 100vh;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  background: var(--liki-paper);
  text-align: center;
  padding: 24px;
}
.fatal h1 { font-size: 22px; margin: 0; }
.fatal .msg { color: var(--liki-text-2); max-width: 440px; margin: 0; }
.actions { display: flex; gap: 12px; margin-top: 8px; }
</style>
