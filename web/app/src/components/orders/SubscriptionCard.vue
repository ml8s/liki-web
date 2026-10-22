<template>
  <el-card
    v-if="subscriptions.length"
    class="app-card subscription-card"
    shadow="never"
  >
    <template #header>
      <h2 class="app-card__title">
        {{ $t('subscriptions.title') }}
      </h2>
    </template>
    <div class="subscription-list">
      <div
        v-for="subscription in subscriptions"
        :key="subscription.id"
        class="subscription"
      >
        <div>
          <div class="subscription-title">
            {{ productName(subscription.product_id) }}
            <el-tag
              :type="subscriptionStatusType(subscription.status)"
              size="small"
            >
              {{ subscriptionStatusLabel(subscription.status) }}
            </el-tag>
            <el-tag
              v-if="subscription.cancel_at_period_end && subscription.status === 'active'"
              type="warning"
              size="small"
            >
              {{ $t('subscriptions.endsAtPeriodEnd') }}
            </el-tag>
          </div>
          <div class="subscription-meta">
            <span v-if="subscription.current_period_end">
              {{ $t('subscriptions.activeUntil') }} {{ formatDateTime(subscription.current_period_end) }}
            </span>
            <span v-if="subscription.next_billing_at">
              {{ $t('subscriptions.nextBilling') }} {{ formatDateTime(subscription.next_billing_at) }}
            </span>
          </div>
        </div>
        <el-button
          v-if="subscription.status === 'active'"
          size="small"
          :loading="updatingId === subscription.id"
          @click="toggle(subscription)"
        >
          {{ subscription.cancel_at_period_end ? $t('subscriptions.resume') : $t('subscriptions.cancel') }}
        </el-button>
      </div>
    </div>
    <div
      v-if="payments.length"
      class="subscription-payments"
    >
      <h3 class="subscription-payments-title">
        {{ $t('subscriptions.paymentHistory') }}
      </h3>
      <div
        v-for="item in payments"
        :key="item.id"
        class="subscription-payment"
      >
        <div>
          <span class="payment-date">{{ formatDateTime(item.created_at) }}</span>
          <span
            v-if="item.period_start && item.period_end"
            class="payment-period"
          >
            {{ formatDateTime(item.period_start) }} – {{ formatDateTime(item.period_end) }}
          </span>
        </div>
        <div class="payment-state">
          <span class="payment-amount">{{ formatMinorUnit(item.amount_minor_units, item.currency_code) }}</span>
          <span
            v-if="item.refunded_minor_units > 0"
            class="payment-refund"
          >
            {{ $t('subscriptions.refundedAmount', { amount: formatMinorUnit(item.refunded_minor_units, item.currency_code) }) }}
          </span>
          <el-tag
            :type="subscriptionPaymentStatusType(item)"
            size="small"
          >
            {{ subscriptionPaymentStatusLabel(item) }}
          </el-tag>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Subscription, SubscriptionPayment } from '@/api/orders'
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'

defineProps<{
  subscriptions: Subscription[]
  payments: SubscriptionPayment[]
  updatingId: string
}>()

const emit = defineEmits<{ toggle: [subscription: Subscription] }>()
const { t } = useI18n()

function toggle(subscription: Subscription) {
  emit('toggle', subscription)
}

function subscriptionStatusLabel(status: Subscription['status']) {
  const keys: Record<Subscription['status'], string> = {
    pending: 'subscriptions.status.pending',
    active: 'subscriptions.status.active',
    on_hold: 'subscriptions.status.onHold',
    cancelled: 'subscriptions.status.cancelled',
    expired: 'subscriptions.status.expired',
    failed: 'subscriptions.status.failed',
  }
  return t(keys[status])
}

function subscriptionStatusType(status: Subscription['status']) {
  if (status === 'active') return 'success'
  if (status === 'pending' || status === 'on_hold') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

function subscriptionPaymentStatusLabel(payment: SubscriptionPayment) {
  if (payment.status === 'refunded') return t('subscriptions.paymentStatus.refunded')
  if (payment.refunded_minor_units > 0) return t('subscriptions.paymentStatus.partiallyRefunded')
  return t('subscriptions.paymentStatus.paid')
}

function subscriptionPaymentStatusType(payment: SubscriptionPayment) {
  if (payment.status === 'refunded') return 'info'
  if (payment.refunded_minor_units > 0) return 'warning'
  return 'success'
}

</script>

<style scoped>
.subscription-list { display: grid; gap: 12px; }
.subscription { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.subscription-title { display: flex; align-items: center; gap: 8px; font-weight: 650; }
.subscription-meta { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 4px; color: var(--liki-text-2); font-size: var(--font-size-label); }
.subscription-payments { margin-top: 18px; padding-top: 14px; border-top: 1px solid var(--liki-line); }
.subscription-payments-title { margin: 0 0 10px; font-size: var(--font-size-body); font-weight: 650; }
.subscription-payment {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 8px 0; border-bottom: 1px solid var(--liki-line);
}
.subscription-payment:last-child { border-bottom: 0; }
.payment-date { font-weight: 600; font-size: var(--font-size-label); }
.payment-period { margin-left: 10px; color: var(--liki-text-2); font-size: var(--font-size-caption); }
.payment-state { display: flex; align-items: center; gap: 10px; }
.payment-amount { font-weight: 600; font-size: var(--font-size-label); }
.payment-refund { color: var(--liki-text-2); font-size: var(--font-size-caption); }
@media (max-width: 640px) {
  .subscription { align-items: flex-start; flex-direction: column; }
  .subscription-payment { align-items: flex-start; flex-direction: column; gap: 6px; }
  .payment-state { flex-wrap: wrap; }
}
</style>
