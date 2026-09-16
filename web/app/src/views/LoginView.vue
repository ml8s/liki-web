<template>
  <main class="login-page">
    <el-card class="login-card" shadow="never">
      <h1>登录 Liki</h1>
      <p class="slogan font-serif">懂命理，用 Liki</p>
      <div class="values">
        <span class="value">确定性排盘</span>
        <span class="value">专业解读</span>
        <span class="value">免费建档</span>
      </div>
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
          <el-input v-model="code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" @update:model-value="maybeAutoVerify" />
        </el-form-item>
        <div class="code-actions">
          <el-button text :disabled="resendIn > 0 || sending" :loading="sending" @click="resend">
            {{ resendIn > 0 ? `重新发送（${resendIn}s）` : '重新发送验证码' }}
          </el-button>
          <el-button text type="primary" @click="unlockEmail">修改邮箱</el-button>
        </div>
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
import { toUserMessage } from '@/utils/errors'
import { onBeforeUnmount, ref } from 'vue'
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
const resendIn = ref(0)
let resendTimer: ReturnType<typeof setInterval> | null = null

function startCountdown() {
  resendIn.value = 60
  if (resendTimer) clearInterval(resendTimer)
  resendTimer = setInterval(() => {
    resendIn.value -= 1
    if (resendIn.value <= 0 && resendTimer) {
      clearInterval(resendTimer)
      resendTimer = null
    }
  }, 1000)
}

async function sendCode() {
  error.value = ''
  sending.value = true
  try {
    const issued = await auth.requestEmailCode(email.value)
    devCode.value = issued.dev_code ?? ''
    codeSent.value = true
    startCountdown()
  } catch (err) {
    error.value = toUserMessage(err, '验证码发送失败')
  } finally {
    sending.value = false
  }
}

function resend() {
  if (resendIn.value > 0 || sending.value) return
  void sendCode()
}

function unlockEmail() {
  codeSent.value = false
  code.value = ''
  error.value = ''
}

function maybeAutoVerify(value: string) {
  // 输满 6 位自动登录（工业化标配），减少一步点击。
  if (value.length === 6 && !verifying.value) void verify()
}

async function verify() {
  if (code.value.length < 6) return
  error.value = ''
  verifying.value = true
  try {
    await auth.verifyEmailCode(email.value, code.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/app'
    await router.push(redirect)
  } catch (err) {
    error.value = toUserMessage(err, '登录失败')
  } finally {
    verifying.value = false
  }
}

onBeforeUnmount(() => {
  if (resendTimer) clearInterval(resendTimer)
})
</script>

<style scoped>
.login-page { min-height: 100vh; display: grid; place-items: center; background: #f7f7f5; padding: 24px; }
.login-card { width: min(420px, 100%); }
h1 { margin: 0 0 8px; font-size: 24px; }
.hint { color: #666; margin: 0 0 20px; }
.slogan { color: var(--liki-primary); margin: 0 0 4px; font-size: 15px; }
.values { display: flex; gap: 8px; justify-content: center; margin: 6px 0 12px; flex-wrap: wrap; }
.value { font-size: 12px; color: var(--liki-text-2); background: var(--liki-primary-tint); padding: 3px 10px; border-radius: 999px; }
.wide { width: 100%; }
.code-actions { display: flex; justify-content: space-between; margin: 0 0 12px; }
.error { margin-top: 16px; }
.dev-code { margin-top: 16px; }
</style>
