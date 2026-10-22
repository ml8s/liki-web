<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="min(420px, calc(100vw - 32px))"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-form
      label-position="top"
      @submit.prevent="submit"
    >
      <el-form-item :label="$t('partner.method')">
        <el-radio-group v-model="method">
          <el-radio-button value="alipay">
            {{ $t('partner.alipay') }}
          </el-radio-button>
          <el-radio-button value="paypal">
            PayPal
          </el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item
        :label="$t('partner.account')"
        required
      >
        <el-input
          v-model="account"
          name="payout-account"
          autocomplete="off"
          spellcheck="false"
          :aria-invalid="!!accountError || undefined"
          :aria-describedby="accountError ? 'payout-account-error' : undefined"
        />
        <p
          v-if="accountError"
          id="payout-account-error"
          class="field-error"
          role="alert"
        >
          {{ accountError }}
        </p>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">
        {{ $t('common.cancel') }}
      </el-button>
      <el-button
        type="primary"
        :loading="submitting"
        @click="submit"
      >
        {{ $t('partner.settlement.requestPayout') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PayoutMethod } from '@/api/partner'

const { t } = useI18n()

const props = defineProps<{
  modelValue: boolean
  title: string
  submitting?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [payload: { method: PayoutMethod; account: string }]
}>()

const method = ref<PayoutMethod>('alipay')
const account = ref('')
const accountError = ref('')

watch(() => props.modelValue, (open) => {
  if (open) {
    method.value = 'alipay'
    account.value = ''
    accountError.value = ''
  }
})

function submit() {
  if (!account.value.trim()) {
    accountError.value = t('partner.accountRequired')
    return
  }
  accountError.value = ''
  emit('submit', { method: method.value, account: account.value.trim() })
}
</script>

<style scoped>
.field-error {
  margin: 6px 0 0;
  color: var(--el-color-danger);
  font-size: var(--font-size-label);
}
</style>
