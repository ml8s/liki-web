<template>
  <section>
    <AppPageHeader title="订单" subtitle="统一 Chat 购买记录。">
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

    <AppEmpty v-else-if="!loading && !orders.length" description="还没有订单" />

    <el-table v-else :data="orders" v-loading="loading" class="orders-table">
      <el-table-column prop="id" label="订单号" min-width="180" show-overflow-tooltip />
      <el-table-column prop="product_id" label="产品" min-width="160" />
      <el-table-column label="金额" width="130">
        <template #default="{ row }">{{ formatAmount(row) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="190" />
      <el-table-column prop="paid_at" label="支付时间" width="190" />
    </el-table>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchChatOrders, type ChatOrder } from '@/api/orders'
import AppAlert from '@/components/app/AppAlert.vue'
import AppEmpty from '@/components/app/AppEmpty.vue'
import AppErrorState from '@/components/app/AppErrorState.vue'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import AppSkeleton from '@/components/app/AppSkeleton.vue'

const orders = ref<ChatOrder[]>([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    orders.value = await fetchChatOrders()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载订单失败'
  } finally {
    loading.value = false
  }
}

function formatAmount(order: ChatOrder) {
  const amount = (order.amount_minor / 100).toFixed(2)
  return `${order.currency} ${amount}`
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

onMounted(load)
</script>

<style scoped>
.orders-table { background: var(--liki-surface); border-radius: var(--el-border-radius-base); }
</style>