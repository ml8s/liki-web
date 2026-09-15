<template>
  <main class="login-page">
    <el-card class="login-card" shadow="never">
      <h1>登录 Liki</h1>
      <p class="slogan font-serif">懂命理，用靈機</p>
      <p class="hint">输入邮箱获取登录验证码。</p>

      <el-form label-position="top" @submit.prevent="sendCode">
        <el-form-item label="邮箱">
          <el-input v-model="email" type="email" autocomplete="email" :disabled="codeSent" />
        </el-form-item>
        <el-button
          v-if="!codeSent"
          type="primary"
          class="wide"
          :loading="sending"
          @click="sendCode"
        >获取验证码</el-button>
      </el-form>

      <el-form v-if="codeSent" label-position="top" @submit.prevent="verify">
        <el-form-item label="验证码">
          <el-input v-model="code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" />
        </el-form-item>
        <el-button type="primary" class="wide" :loading="verifying" @click="verify">登录</el-button>
      </el-form>

      <el-alert
        v-if="devCode"
        title="开发模式验证码"
        :description="devCode"
        type="info"
        show-icon
        :closable="false"
        class="dev-code"
      />

      <el-alert
        v-if="error"
        :title="error"
        type="error"
        show-icon
        :closable="false"
        class="error"
      />
    </el-card>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const email = ref('')
const code = ref('')
const codeSent = ref(false)
const devCode = ref('')
const sending = ref(false)
const verifying = ref(false)
const error = ref('')

async function sendCode() {
  error.value = ''
  sending.value = true
  try {
    const issued = await auth.requestEmailCode(email.value)
    devCode.value = issued.dev_code ?? ''
    codeSent.value = true
  } catch (err) {
    error.value = err instanceof Error ? err.message : '验证码发送失败'
  } finally {
    sending.value = false
  }
}

async function verify() {
  error.value = ''
  verifying.value = true
  try {
    await auth.verifyEmailCode(email.value, code.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app'
    await router.push(redirect)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败'
  } finally {
    verifying.value = false
  }
}
</script>

<style scoped>
.login-page { min-height: 100vh; display: grid; place-items: center; background: #f7f7f5; padding: 24px; }
.login-card { width: min(420px, 100%); }
h1 { margin: 0 0 8px; font-size: 24px; }
.hint { color: #666; margin: 0 0 20px; }
.slogan { color: var(--liki-primary); margin: 0 0 4px; font-size: 15px; }
.wide { width: 100%; }
.error { margin-top: 16px; }
.dev-code { margin-top: 16px; }
</style>
