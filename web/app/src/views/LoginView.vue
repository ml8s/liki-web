<template>
  <main class="login-page">
    <section class="login-brand">
      <div class="brand-mark">Liki<span>·</span>{{ $t('auth.brandTail') }}</div>
      <p class="brand-headline">{{ $t('auth.headline') }}</p>
      <p class="brand-description">
        {{ $t('auth.description') }}
      </p>
      <ul>
        <li>{{ $t('auth.valueChart') }}</li>
        <li>{{ $t('auth.valueInterpretation') }}</li>
        <li>{{ $t('auth.valueProfile') }}</li>
      </ul>
    </section>

    <section class="login-form-side">
    <el-card class="login-card" shadow="never">
      <h1>{{ $t('auth.loginTitle') }}</h1>
      <p class="slogan font-serif">{{ $t('auth.slogan') }}</p>
      <div class="values">
        <span class="value">{{ $t('auth.valueChart') }}</span>
        <span class="value">{{ $t('auth.valueInterpretation') }}</span>
        <span class="value">{{ $t('auth.valueProfile') }}</span>
      </div>
      <p class="hint">{{ $t('auth.codeHint') }}</p>

      <el-form label-position="top" @submit.prevent="sendCode">
        <el-form-item :label="$t('auth.email')">
          <el-input
            v-model="email"
            type="email"
            name="email"
            autocomplete="email"
            spellcheck="false"
            :aria-invalid="!!error || undefined"
            :aria-describedby="error ? 'login-error' : undefined"
            :disabled="codeSent"
          />
        </el-form-item>
        <el-button
          v-if="!codeSent"
          native-type="submit"
          type="primary"
          class="wide"
          :loading="sending"
        >{{ $t('auth.getCode') }}</el-button>
      </el-form>

      <el-form v-if="codeSent" label-position="top" @submit.prevent="verify">
        <el-form-item :label="$t('auth.verificationCode')">
          <el-input
            v-model="code"
            inputmode="numeric"
            name="verification-code"
            autocomplete="one-time-code"
            spellcheck="false"
            maxlength="6"
            :placeholder="$t('auth.codePlaceholder')"
            :aria-invalid="!!error || undefined"
            :aria-describedby="error ? 'login-error' : undefined"
            @update:model-value="maybeAutoVerify"
          />
        </el-form-item>
        <div class="code-actions">
          <el-button text :disabled="resendIn > 0 || sending" :loading="sending" @click="resend">
            {{ resendIn > 0 ? $t('auth.resendIn', { seconds: resendIn }) : $t('auth.resend') }}
          </el-button>
          <el-button text type="primary" @click="unlockEmail">{{ $t('auth.changeEmail') }}</el-button>
        </div>
        <el-button type="primary" class="wide" native-type="submit" :loading="verifying">{{ $t('auth.login') }}</el-button>
      </el-form>

      <el-alert
        v-if="devCode"
        :title="$t('auth.devCode')"
        :description="devCode"
        type="info"
        show-icon
        :closable="false"
        class="dev-code"
      />

      <el-alert
        v-if="error"
        id="login-error"
        :title="error"
        type="error"
        show-icon
        :closable="false"
        class="error"
      />
    </el-card>
    </section>
  </main>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

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
    error.value = toUserMessage(err, t('auth.sendCodeFailed'))
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
    const requested = typeof route.query.redirect === 'string' ? route.query.redirect : '/app'
    await router.push(safeRedirect(requested))
  } catch (err) {
    error.value = toUserMessage(err, t('auth.loginFailed'))
  } finally {
    verifying.value = false
  }
}

function safeRedirect(value: string) {
  if (!value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\') || value.includes('://')) {
    return '/app'
  }
  return value
}

onBeforeUnmount(() => {
  if (resendTimer) clearInterval(resendTimer)
})
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  min-height: 100dvh;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 520px);
  align-items: center;
  gap: 48px;
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 24px;
  background:
    radial-gradient(60% 70% at 10% 0%, rgba(83, 58, 253, 0.10), transparent 65%),
    var(--liki-paper);
}
.login-brand { padding: 24px 0; }
.brand-mark {
  margin-bottom: 28px;
  color: var(--liki-ink);
  font-family: var(--liki-font-serif);
  font-size: 24px;
  font-weight: 700;
}
.brand-mark span { color: var(--liki-primary); font-style: normal; }
.login-brand .brand-description {
  max-width: 390px;
  margin: 0 0 28px;
  color: var(--liki-text-2);
  font-size: 16px;
  line-height: 1.7;
  text-wrap: pretty;
}
.login-brand ul {
  display: grid;
  gap: 10px;
  padding: 0;
  margin: 0;
  list-style: none;
  color: var(--liki-ink);
}
.login-brand li {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
}
.login-brand li::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--liki-primary);
}
.login-form-side { display: grid; place-items: center; }
.login-card {
  width: min(420px, 100%);
  border: 1px solid var(--liki-line);
  border-top: 3px solid var(--liki-primary);
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-card);
}
h1 { margin: 0 0 8px; font-size: 26px; font-weight: 650; letter-spacing: -0.03em; line-height: 1.2; }
.brand-headline {
  max-width: 420px;
  margin: 0 0 16px;
  color: var(--liki-ink);
  font-size: clamp(30px, 4vw, 44px);
  font-weight: 650;
  line-height: 1.1;
  letter-spacing: -0.04em;
}
.hint { color: var(--liki-text-2); margin: 0 0 20px; }
.slogan { color: var(--liki-primary); margin: 0 0 4px; font-size: 15px; }
.values { display: flex; gap: 8px; justify-content: flex-start; margin: 6px 0 12px; flex-wrap: wrap; }
.value { font-size: 12px; color: var(--liki-text-2); background: var(--liki-primary-tint); padding: 3px 10px; border-radius: 999px; }
.wide { width: 100%; }
.code-actions { display: flex; justify-content: space-between; margin: 0 0 12px; }
.error { margin-top: 16px; }
.dev-code { margin-top: 16px; }

@media (max-width: 900px) {
  .login-page {
    grid-template-columns: 1fr;
    gap: 28px;
    place-items: center;
    padding: 24px 16px;
  }
  .login-brand { display: none; }
}
</style>
