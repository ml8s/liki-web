<template>
  <section class="page-stack">
    <AppPageHeader
      :title="$t('router.orders')"
      :subtitle="$t('orders.subtitle')"
    >
      <template #actions>
        <el-button
          v-if="hasData"
          :loading="loading"
          @click="load"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-result
      v-if="error && !hasData"
      icon="error"
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

    <el-skeleton
      v-else-if="loading && !hasData"
      :rows="6"
      animated
      aria-busy="true"
    />

    <AppEmptyState
      v-else-if="!loading && !hasData"
      :description="$t('orders.empty')"
    >
      <el-button
        type="primary"
        tag="router-link"
        to="/chat"
      >
        {{ $t('orders.unlock') }}
      </el-button>
    </AppEmptyState>

    <!-- 订阅与一次性订单是并列区块：存在订阅时订单列表仍需渲染。 -->
    <template v-else>
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        :closable="false"
        show-icon
      />

      <SubscriptionCard
        :subscriptions="subscriptions"
        :payments="subscriptionPayments"
        :updating-id="updatingId"
        @toggle="toggleCancellation"
      />

      <el-alert
        v-if="pendingOrders.length"
        :title="$t('orders.pendingTitle')"
        :description="$t('orders.pendingBody')"
        type="warning"
        show-icon
        :closable="false"
      />

      <!-- 桌面：表格 -->
      <el-table
        v-if="!isMobile"
        v-loading="loading"
        :aria-busy="loading"
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
          width="150"
        >
          <template #default="{ row }">
            <div class="status-cell">
              <el-tag
                :type="statusType(row.status)"
                size="small"
              >
                {{ statusLabel(row.status) }}
              </el-tag>
              <el-tag
                v-if="intentBadge(asOrder(row))"
                type="info"
                size="small"
              >
                {{ intentBadge(asOrder(row)) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          :label="$t('orders.createdAt')"
          width="170"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column
          :label="$t('orders.paidAt')"
          width="170"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.paid_at) }}
          </template>
        </el-table-column>
        <el-table-column
          width="200"
          align="right"
          fixed="right"
        >
          <template #default="{ row }">
            <RecordActions
              :actions="orderActions(asOrder(row))"
              @action="key => onOrderAction(key, asOrder(row))"
            />
          </template>
        </el-table-column>
      </el-table>

      <!-- 移动端：卡片降级 -->
      <div
        v-else
        v-loading="loading"
        :aria-busy="loading"
        class="order-cards"
      >
        <div
          v-for="order in orders"
          :key="order.id"
          class="order-card"
        >
          <div class="order-top">
            <span class="order-id">{{ order.id }}</span>
            <span class="order-tags">
              <el-tag
                :type="statusType(order.status)"
                size="small"
              >
                {{ statusLabel(order.status) }}
              </el-tag>
              <el-tag
                v-if="intentBadge(order)"
                type="info"
                size="small"
              >
                {{ intentBadge(order) }}
              </el-tag>
            </span>
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
          <RecordActions
            class="order-actions"
            :actions="orderActions(order)"
            @action="key => onOrderAction(key, order)"
          />
        </div>
      </div>
    </template>

    <el-drawer
      v-model="detailOpen"
      :title="$t('orders.detail.title')"
      :size="isMobile ? '92%' : '420px'"
    >
      <template v-if="detailData">
        <div class="detail-tags">
          <el-tag
            :type="statusType(detailData.order.status)"
            size="small"
          >
            {{ statusLabel(detailData.order.status) }}
          </el-tag>
          <el-tag
            v-if="intentBadge(detailData.order)"
            type="info"
            size="small"
          >
            {{ intentBadge(detailData.order) }}
          </el-tag>
        </div>
        <div class="order-row detail-row">
          <span class="k">{{ $t('orders.id') }}</span><span class="detail-id">{{ detailData.order.id }}</span>
        </div>
        <div class="order-row detail-row">
          <span class="k">{{ $t('orders.product') }}</span><span>{{ productName(detailData.order.product_id) }}</span>
        </div>
        <div class="order-row detail-row">
          <span class="k">{{ $t('orders.amount') }}</span><span>{{ formatMinorUnit(detailData.order.amount_minor, detailData.order.currency) }}</span>
        </div>
        <div
          v-if="detailData.payment"
          class="order-row detail-row"
        >
          <span class="k">{{ $t('orders.detail.payment') }}</span><span>{{ providerLabel(detailData.payment.provider) }}</span>
        </div>
        <div
          v-if="detailData.payment"
          class="order-row detail-row"
        >
          <span class="k">{{ $t('orders.detail.paymentStatus') }}</span><span>{{ intentStatusLabel(detailData.payment.status) }}</span>
        </div>
        <div
          v-if="detailData.order.paid_at"
          class="order-row detail-row"
        >
          <span class="k">{{ $t('orders.paidShort') }}</span><span>{{ formatDateTime(detailData.order.paid_at) }}</span>
        </div>
        <div
          v-if="detailData.payment && detailData.payment.refunded_amount_minor > 0"
          class="order-row detail-row"
        >
          <span class="k">{{ $t('orders.detail.refund') }}</span><span>{{ $t('orders.detail.refundedAmount', { amount: formatMinorUnit(detailData.payment.refunded_amount_minor, detailData.order.currency) }) }}</span>
        </div>
        <div
          v-if="detailData.order.status === 'paid'"
          class="order-row detail-row"
        >
          <span class="k">{{ $t('orders.detail.entitlement') }}</span>
          <span v-if="activeEntitlement">{{ $t('orders.detail.entitlementActive', { date: formatDateTime(activeEntitlement.expires_at) }) }}</span>
          <span v-else>{{ $t('orders.detail.entitlementProcessing') }}</span>
        </div>

        <el-timeline class="detail-timeline">
          <el-timeline-item
            :timestamp="formatDateTime(detailData.order.created_at)"
            type="primary"
          >
            {{ $t('orders.detail.nodeCreated') }}
          </el-timeline-item>
          <el-timeline-item
            v-if="detailData.payment?.paid_at"
            :timestamp="formatDateTime(detailData.payment.paid_at)"
            type="success"
          >
            {{ $t('orders.detail.nodePaid') }}
          </el-timeline-item>
          <el-timeline-item
            v-else-if="detailData.order.status === 'pending'"
            :timestamp="formatDateTime(detailData.order.updated_at)"
          >
            {{ $t('orders.detail.nodeWaiting') }}
          </el-timeline-item>
          <el-timeline-item
            v-if="detailData.payment?.refunded_at"
            :timestamp="formatDateTime(detailData.payment.refunded_at)"
            type="info"
          >
            {{ $t('orders.detail.nodeRefunded') }}
          </el-timeline-item>
          <el-timeline-item
            v-if="detailData.order.status === 'canceled'"
            :timestamp="formatDateTime(detailData.order.updated_at)"
            type="info"
          >
            {{ $t('orders.statusCanceled') }}
          </el-timeline-item>
        </el-timeline>
      </template>
      <el-skeleton
        v-else-if="detailLoading"
        :rows="4"
        animated
      />
      <p
        v-else
        class="detail-error"
      >
        {{ detailError }}
      </p>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { confirmAction } from '@/utils/dialogs'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { cancelOrder as apiCancelOrder, fetchOrderDetail, fetchOrders, fetchSubscriptionPayments, fetchSubscriptions, resumeOrderCheckout, setSubscriptionCancellation, type Order, type OrderDetail, type Subscription, type SubscriptionPayment } from '@/api/orders'
import { ApiError } from '@/api/http'
import { useEntitlementsStore } from '@/stores/entitlements'
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'
import { useMobile } from '@/composables/useMobile'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppEmptyState from '@/components/app/AppEmptyState.vue'
import SubscriptionCard from '@/components/orders/SubscriptionCard.vue'
import RecordActions, { type RecordAction } from '@/components/app/RecordActions.vue'

const orders = ref<Order[]>([])
const subscriptions = ref<Subscription[]>([])
const subscriptionPayments = ref<SubscriptionPayment[]>([])
const updatingId = ref('')
const actingId = ref('')
const loading = ref(false)
const error = ref('')
const isMobile = useMobile()
const { t, te } = useI18n()
const entitlements = useEntitlementsStore()

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailData = ref<OrderDetail | null>(null)
const detailError = ref('')

const activeEntitlement = computed(() =>
  entitlements.items.find(item => item.status === 'active' && item.skills.includes('liki')) ?? null,
)

const hasData = computed(() => orders.value.length > 0 || subscriptions.value.length > 0)

const pendingOrders = computed(() => orders.value.filter(order => order.status === 'pending'))

/** 行操作标准：每个订单都可打开详情，再按 订单态×intent 态 派生可用动作。 */
function orderActions(order: Order): RecordAction[] {
  return [{ key: 'detail', label: t('orders.detail.action') }, ...statusActions(order)]
}

function statusActions(order: Order): RecordAction[] {
  if (order.status === 'pending') {
    if (order.intent_status === 'expired') {
      // 已过期的订单仍可取消：过期也是未完结，用户需要能整理列表。
      return [
        { key: 'repurchase', label: t('orders.repurchase'), to: '/chat' },
        { key: 'cancel', label: t('orders.cancel'), danger: true },
      ]
    }
    if (order.intent_status === 'refunded') {
      return [{ key: 'repurchase', label: t('orders.repurchase'), to: '/chat' }]
    }
    if (order.intent_status === 'paid') {
      return [{ key: 'refresh', label: t('orders.refresh') }]
    }
    return [
      { key: 'resume', label: t('orders.resume') },
      { key: 'cancel', label: t('orders.cancel'), danger: true },
    ]
  }
  if (order.status === 'paid') {
    return [{ key: 'use', label: t('orders.use'), to: '/chat' }]
  }
  return [{ key: 'repurchase', label: t('orders.repurchase'), to: '/chat' }]
}

function intentBadge(order: Order): string {
  if (order.intent_status === 'refunded') return t('orders.statusRefunded')
  if (order.status === 'pending' && order.intent_status === 'expired') return t('orders.statusExpired')
  return ''
}

function onOrderAction(key: string, order: Order) {
  if (key === 'detail') void openDetail(order)
  else if (key === 'resume') void resumeCheckout(order)
  else if (key === 'cancel') void confirmCancelOrder(order)
  else if (key === 'refresh') void load()
}

function providerLabel(provider: string): string {
  const key = `orders.detail.provider.${provider}`
  return te(key) ? t(key) : provider
}

function intentStatusLabel(status: Order['intent_status']): string {
  return status ? t(`orders.intentStatus.${status}`) : '—'
}

async function openDetail(order: Order) {
  detailOpen.value = true
  detailLoading.value = true
  detailData.value = null
  detailError.value = ''
  try {
    detailData.value = await fetchOrderDetail(order.id)
  } catch (err) {
    detailError.value = toUserMessage(err, t('orders.detail.loadFailed'))
  } finally {
    detailLoading.value = false
  }
}

async function resumeCheckout(order: Order) {
  if (actingId.value) return
  actingId.value = order.id
  error.value = ''
  try {
    const result = await resumeOrderCheckout(order.id)
    // 与购买同规则：只跟随相对路径或 https 支付域，防开放重定向。
    const url = result.checkout_url
    if (!url.startsWith('/') && !url.startsWith('https://')) {
      error.value = t('orders.resumeFailed')
      return
    }
    window.location.href = url
  } catch (err) {
    if (err instanceof ApiError && err.code === 'order_expired') order.intent_status = 'expired'
    if (err instanceof ApiError && err.code === 'order_already_paid') await load()
    error.value = toUserMessage(err, t('orders.resumeFailed'))
  } finally {
    actingId.value = ''
  }
}

async function confirmCancelOrder(order: Order) {
  if (!await confirmAction({
    title: t('orders.cancelTitle'),
    message: t('orders.cancelConfirm'),
    confirmText: t('orders.cancel'),
  })) return
  if (actingId.value) return
  actingId.value = order.id
  error.value = ''
  try {
    await apiCancelOrder(order.id)
    order.status = 'canceled'
  } catch (err) {
    if (err instanceof ApiError && err.code === 'order_not_pending') await load()
    error.value = toUserMessage(err, t('orders.cancelFailed'))
  } finally {
    actingId.value = ''
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [orderItems, subscriptionItems, paymentItems] = await Promise.all([
      fetchOrders(),
      fetchSubscriptions(),
      fetchSubscriptionPayments(),
      entitlements.load(),
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

async function toggleCancellation(subscription: Subscription) {
  if (!subscription.cancel_at_period_end) {
    if (!await confirmAction({
      title: t('subscriptions.cancelConfirmTitle'),
      message: t('subscriptions.cancelConfirmBody'),
      confirmText: t('subscriptions.cancel'),
    })) return
  }
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
.orders-table { background: var(--liki-surface); border-radius: var(--radius-control); }
.orders-table, .subscription-meta, .payment-amount, .payment-date, .order-card { font-variant-numeric: tabular-nums; }
.order-cards { display: grid; gap: 12px; }
.order-card {
  background: var(--liki-surface); border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg); padding: 14px 16px;
}
.order-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 8px; }
.order-tags { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; flex-shrink: 0; }
.status-cell { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.order-actions { margin-top: 10px; padding-top: 8px; border-top: 1px solid var(--liki-line); justify-content: flex-start; }
.detail-tags { display: flex; align-items: center; gap: 6px; margin-bottom: 12px; }
.order-row.detail-row .k { width: 5em; }
.detail-id { word-break: break-all; }
.detail-timeline { margin-top: 16px; }
.detail-error { color: var(--liki-text-2); }
.order-id { font-weight: 600; font-size: var(--font-size-label); word-break: break-all; }
.order-row { display: flex; gap: 8px; font-size: var(--font-size-body); color: var(--liki-ink); padding: 3px 0; }
.order-row .k { color: var(--liki-text-2); width: 40px; flex-shrink: 0; }
</style>
