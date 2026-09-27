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
      v-if="error && !orders.length"
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
      v-else-if="loading && !orders.length"
      :rows="6"
      animated
      aria-busy="true"
    />

    <el-empty
      v-else-if="!loading && !orders.length"
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
import { fetchChatOrders, type ChatOrder } from '@/api/orders'
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'
import { useMobile } from '@/composables/useMobile'
import AppPageHeader from '@/components/app/AppPageHeader.vue'

const orders = ref<ChatOrder[]>([])
const loading = ref(false)
const error = ref('')
const isMobile = useMobile()
const { t } = useI18n()

const pendingOrders = computed(() => orders.value.filter(order => order.status === 'pending'))

async function load() {
  loading.value = true
  error.value = ''
  try {
    orders.value = await fetchChatOrders()
  } catch (err) {
    error.value = toUserMessage(err, t('orders.loadFailed'))
  } finally {
    loading.value = false
  }
}

function formatAmount(row: Record<string, unknown>) {
  const order = row as unknown as ChatOrder
  return formatMinorUnit(order.amount_minor, order.currency)
}

function statusLabel(status: ChatOrder['status']) {
  const labels = { pending: t('orders.statusPending'), paid: t('orders.statusPaid'), canceled: t('orders.statusCanceled') }
  return labels[status]
}

function statusType(status: ChatOrder['status']) {
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
