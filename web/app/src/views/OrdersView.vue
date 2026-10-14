<template>
  <section>
    <AppPageHeader
      :title="$t('router.orders')"
      :subtitle="$t('orders.subtitle')"
    >
      <template #actions>
        <el-button
          :loading="loading"
          @click="load"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-result
      v-if="error && !orders.length && !subscriptions.length"
      icon="warning"
      :title="$t('orders.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="load"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>
    <el-alert
      v-else-if="error && orders.length"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <el-skeleton
      v-else-if="loading && !orders.length && !subscriptions.length"
      :rows="6"
      animated
      aria-busy="true"
    />

    <el-empty
      v-else-if="!loading && !orders.length && !subscriptions.length"
      :description="$t('orders.empty')"
    >
      <el-button
        type="primary"
        tag="a"
        to="/chat"
      >
        {{ $t('orders.unlock') }}
      </el-button>
    </el-empty>

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
          <div class="subscription-main">
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
            v-if="subscription.status === 'active' && subscription.provider_subscription_id"
            size="small"
            :loading="updatingId === subscription.id"
            @click="toggleCancellation(subscription)"
          >
            {{ subscription.cancel_at_period_end ? $t('subscriptions.resume') : $t('subscriptions.cancel') }}
          </el-button>
        </div>
      </div>
      <div
        v-if="subscriptionPayments.length"
        class="subscription-payments"
      >
        <h3 class="subscription-payments-title">
          {{ $t('subscriptions.paymentHistory') }}
        </h3>
        <div
          v-for="item in subscriptionPayments"
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

    <!-- 桌面：表格 -->
    <el-table
      v-else-if="!isMobile"
      v-loading="loading"
      :data="orders"
      class="orders-table"
    >
      <el-table-column
        prop="id"
        :label="$t('orders.id')"
        min-width="180"
        show-overflow-tooltip
      />
      <el-table-column
        :label="$t('orders.product')"
        min-width="160"
      >
        <template #default="{ row }">
          {{ productName(row.product_id) }}
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('orders.amount')"
        width="130"
      >
        <template #default="{ row }">
          {{ formatAmount(row) }}
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('orders.status')"
        width="120"
      >
        <template #default="{ row }">
          <el-tag
            :type="statusType(row.status)"
            size="small"
          >
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('orders.createdAt')"
        width="190"
      >
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column
        :label="$t('orders.paidAt')"
        width="190"
      >
        <template #default="{ row }">
          {{ formatDateTime(row.paid_at) }}
        </template>
      </el-table-column>
    </el-table>

    <!-- 移动端：卡片降级 -->
    <div
      v-else
      v-loading="loading"
      class="order-cards"
    >
      <div
        v-for="order in orders"
        :key="order.id"
        class="order-card"
      >
        <div class="order-top">
          <span class="order-id">{{ order.id }}</span>
          <el-tag
            :type="statusType(order.status)"
            size="small"
          >
            {{ statusLabel(order.status) }}
          </el-tag>
        </div>
        <div class="order-row">
          <span class="k">{{ $t('orders.product') }}</span><span>{{ productName(order.product_id) }}</span>
        </div>
        <div class="order-row">
          <span class="k">{{ $t('orders.amount') }}</span><span>{{ formatAmount(order) }}</span>
        </div>
        <div class="order-row">
          <span class="k">{{ $t('orders.createdShort') }}</span><span>{{ formatDateTime(order.created_at) }}</span>
        </div>
        <div
          v-if="order.paid_at"
          class="order-row"
        >
          <span class="k">{{ $t('orders.paidShort') }}</span><span>{{ formatDateTime(order.paid_at) }}</span>
        </div>
      </div>
    </div>
    <el-alert
      v-if="pendingOrders.length"
      :title="$t('orders.pendingTitle')"
      :description="$t('orders.pendingBody')"
      type="warning"
      show-icon
      :closable="false"
      class="pending-alert"
    />
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { fetchOrders, fetchSubscriptionPayments, fetchSubscriptions, setSubscriptionCancellation, type Order, type Subscription, type SubscriptionPayment } from '@/api/orders'
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'
import { useMobile } from '@/composables/useMobile'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const orders = ref<Order[]>([])
const subscriptions = ref<Subscription[]>([])
const subscriptionPayments = ref<SubscriptionPayment[]>([])
const updatingId = ref('')
const loading = ref(false)
const error = ref('')
const isMobile = useMobile()
const { t } = useI18n()

const pendingOrders = computed(() => orders.value.filter(order => order.status === 'pending'))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [orderItems, subscriptionItems, paymentItems] = await Promise.all([
      fetchOrders(),
      fetchSubscriptions(),
      fetchSubscriptionPayments(),
    ])
    orders.value = orderItems
    subscriptions.value = subscriptionItems
    subscriptionPayments.value = paymentItems
  } catch (err) {
    error.value = toUserMessage(err, t('orders.loadFailed'))
  } finally {
    loading.value = false
  }
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

async function toggleCancellation(subscription: Subscription) {
  updatingId.value = subscription.id
  error.value = ''
  try {
    const cancel = !subscription.cancel_at_period_end
    subscription.cancel_at_period_end = await setSubscriptionCancellation(subscription.id, cancel)
  } catch (err) {
    error.value = toUserMessage(err, t('subscriptions.updateFailed'))
  } finally {
    updatingId.value = ''
  }
}

function asOrder(row: Record<string, unknown>): Order {
  return row as unknown as Order
}

function formatAmount(row: Record<string, unknown>) {
  const order = asOrder(row)
  return formatMinorUnit(order.amount_minor, order.currency)
}

function statusLabel(status: Order['status']) {
  const labels = { pending: t('orders.statusPending'), paid: t('orders.statusPaid'), canceled: t('orders.statusCanceled') }
  return labels[status]
}

function statusType(status: Order['status']) {
  if (status === 'paid') return 'success'
  if (status === 'canceled') return 'info'
  return 'warning'
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.pending-alert { margin-bottom: 16px; }
.subscription-card { margin-bottom: 16px; }
.subscription-list { display: grid; gap: 12px; }
.subscription { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.subscription-title { display: flex; align-items: center; gap: 8px; font-weight: 650; }
.subscription-meta { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 4px; color: var(--liki-text-2); font-size: 13px; }
.subscription-payments { margin-top: 18px; padding-top: 14px; border-top: 1px solid var(--liki-line); }
.subscription-payments-title { margin: 0 0 10px; font-size: 14px; font-weight: 650; }
.subscription-payment {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 8px 0; border-bottom: 1px solid var(--liki-line);
}
.subscription-payment:last-child { border-bottom: 0; }
.payment-date { font-weight: 600; font-size: 13px; }
.payment-period { margin-left: 10px; color: var(--liki-text-2); font-size: 12px; }
.payment-state { display: flex; align-items: center; gap: 10px; }
.payment-amount { font-weight: 600; font-size: 13px; }
.payment-refund { color: var(--liki-text-2); font-size: 12px; }
@media (max-width: 640px) { .subscription { align-items: flex-start; flex-direction: column; } }
.subscription-payment { align-items: flex-start; flex-direction: column; gap: 6px; }
.payment-state { flex-wrap: wrap; }
.orders-table { background: var(--liki-surface); border-radius: var(--el-border-radius-base); }
.order-cards { display: grid; gap: 12px; }
.order-card {
  background: var(--liki-surface); border: 1px solid var(--liki-line);
  border-radius: 12px; padding: 14px 16px;
}
.order-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.order-id { font-weight: 600; font-size: 13px; word-break: break-all; }
.order-row { display: flex; gap: 8px; font-size: 14px; color: var(--liki-ink); padding: 3px 0; }
.order-row .k { color: var(--liki-text-2); width: 40px; flex-shrink: 0; }
</style>
