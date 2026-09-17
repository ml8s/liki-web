<template>
  <section>
    <AppPageHeader title="订单" subtitle="Liki Chat 购买记录。">
      <template #actions>
        <el-button :loading="loading" @click="load">刷新</el-button>
      </template>
    </AppPageHeader>

    <AppErrorState
      v-if="error && !orders.length"
      title="订单加载失败"
      :description="error"
      @retry="load"
    />
    <AppAlert v-else-if="error && orders.length" :title="error" type="error" />

    <AppSkeleton v-if="loading && !orders.length" :rows="6" />

    <AppEmpty v-else-if="!loading && !orders.length" description="还没有订单，解锁 Chat 后即可开始" />

    <!-- 桌面：表格 -->
    <el-table v-else-if="!isMobile" :data="orders" v-loading="loading" class="orders-table">
      <el-table-column prop="id" label="订单号" min-width="180" show-overflow-tooltip />
      <el-table-column label="产品" min-width="160">
        <template #default="{ row }">{{ productName(row.product_id) }}</template>
      </el-table-column>
      <el-table-column label="金额" width="130">
        <template #default="{ row }">{{ formatAmount(row) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="190">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="支付时间" width="190">
        <template #default="{ row }">{{ formatDateTime(row.paid_at) }}</template>
      </el-table-column>
    </el-table>

    <!-- 移动端：卡片降级 -->
    <div v-else class="order-cards" v-loading="loading">
      <div v-for="order in orders" :key="order.id" class="order-card">
        <div class="order-top">
          <span class="order-id">{{ order.id }}</span>
          <el-tag :type="statusType(order.status)" size="small">{{ statusLabel(order.status) }}</el-tag>
        </div>
        <div class="order-row"><span class="k">产品</span><span>{{ productName(order.product_id) }}</span></div>
        <div class="order-row"><span class="k">金额</span><span>{{ formatAmount(order) }}</span></div>
        <div class="order-row"><span class="k">创建</span><span>{{ formatDateTime(order.created_at) }}</span></div>
        <div v-if="order.paid_at" class="order-row"><span class="k">支付</span><span>{{ formatDateTime(order.paid_at) }}</span></div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { toUserMessage } from '@/utils/errors'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { fetchChatOrders, type ChatOrder } from '@/api/orders'
import { formatDateTime, productName } from '@/utils/format'
import AppAlert from '@/components/app/AppAlert.vue'
import AppEmpty from '@/components/app/AppEmpty.vue'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const orders = ref<ChatOrder[]>([])
const loading = ref(false)
const error = ref('')
const isMobile = ref(false)
let mq: MediaQueryList | null = null

function onMqChange(e: MediaQueryListEvent | MediaQueryList) {
  isMobile.value = e.matches
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    orders.value = await fetchChatOrders()
  } catch (err) {
    error.value = toUserMessage(err, '加载订单失败')
  } finally {
    loading.value = false
  }
}

function formatAmount(row: unknown) {
  const order = row as ChatOrder
  const amount = (order.amount_minor / 100).toFixed(2)
  const symbol = order.currency === 'CNY' ? '¥' : order.currency === 'USD' ? '$' : `${order.currency} `
  return `${symbol}${amount}`
}

function statusLabel(status: ChatOrder['status']) {
  const labels = { pending: '待支付', paid: '已支付', canceled: '已取消' }
  return labels[status]
}

function statusType(status: ChatOrder['status']) {
  if (status === 'paid') return 'success'
  if (status === 'canceled') return 'info'
  return 'warning'
}

onMounted(() => {
  mq = window.matchMedia('(max-width: 640px)')
  onMqChange(mq)
  mq.addEventListener('change', onMqChange)
  load()
})

onBeforeUnmount(() => {
  mq?.removeEventListener('change', onMqChange)
})
</script>

<style scoped>
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