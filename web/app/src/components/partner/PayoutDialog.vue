<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="420px"
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
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">
        {{ $t('common.cancel') }}
      </el-button>
      <el-button
        type="primary"
        :loading="submitting"
        native-type="submit"
        @click="submit"
      >
        {{ $t('partner.settlement.requestPayout') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
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

watch(() => props.modelValue, (open) => {
  if (open) {
    method.value = 'alipay'
    account.value = ''
  }
})

function submit() {
  if (!account.value.trim()) {
    ElMessage.warning(t('partner.accountRequired'))
    return
  }
  emit('submit', { method: method.value, account: account.value.trim() })
}
</script>
