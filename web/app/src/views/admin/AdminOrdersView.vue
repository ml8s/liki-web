<template>
  <section>
    <AdminPageHeader title="订单">
      <template #actions>
        <el-button
          :loading="orders.loading.value"
          @click="reloadCurrentPage"
        >
          刷新
        </el-button>
      </template>
    </AdminPageHeader>

    <el-form
      inline
      @submit.prevent="submitOrderFilters"
    >
      <el-form-item label="状态">
        <el-select
          v-model="draft.status"
          class="admin-filter-control"
        >
          <el-option
            v-for="option in statusOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="搜索">
        <el-input
          v-model="draft.search"
          placeholder="订单号 / 邮箱 / 支付凭证"
          clearable
          spellcheck="false"
          class="admin-search-control"
        />
      </el-form-item>
      <el-button
        type="primary"
        native-type="submit"
      >
        查询
      </el-button>
    </el-form>

    <el-alert
      v-if="orders.error.value"
      :title="orders.error.value"
      type="error"
      show-icon
      :closable="false"
      class="query-error"
    />

    <div
      v-loading="orders.loading.value"
      :aria-busy="orders.loading.value"
    >
      <el-table
        v-if="!isMobile"
        :data="orders.data.value"
        row-key="id"
      >
        <el-table-column
          prop="id"
          label="订单号"
          min-width="190"
          show-overflow-tooltip
        />
        <el-table-column
          prop="email"
          label="用户邮箱"
          min-width="200"
          show-overflow-tooltip
        />
        <el-table-column
          prop="product_id"
          label="产品"
          min-width="130"
        />
        <el-table-column
          label="金额"
          width="140"
        >
          <template #default="{ row }">
            {{ formatAdminMoney(row.amount_minor, row.currency) }}
          </template>
        </el-table-column>
        <el-table-column
          label="状态"
          width="110"
        >
          <template #default="{ row }">
            <AdminStatusTag
              domain="order"
              :status="row.status"
            />
          </template>
        </el-table-column>
        <el-table-column
          label="支付凭证"
          min-width="180"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span class="mono muted">{{ row.payment_intent_id || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="创建时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatAdminDateTime(row.created_at) || '-' }}
          </template>
        </el-table-column>
        <el-table-column
          label="支付时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatAdminDateTime(row.paid_at) || '-' }}
          </template>
        </el-table-column>
      </el-table>

      <div
        v-if="isMobile && !orders.loading.value && !orders.data.value.length"
      >
        <el-empty description="暂无订单" />
      </div>
      <div v-if="isMobile">
        <AdminRecordCard
          v-for="row in orders.data.value"
          :key="row.id"
          status-domain="order"
          :fields="[
            { key: 'id', label: '订单号', value: row.id, kind: 'mono' },
            { key: 'email', label: '用户邮箱', value: row.email },
            { key: 'product', label: '产品', value: row.product_id },
            { key: 'amount', label: '金额', value: row.amount_minor, kind: 'money', currency: row.currency },
            { key: 'status', label: '状态', value: row.status, kind: 'status' },
            { key: 'intent', label: '支付凭证', value: row.payment_intent_id, kind: 'mono' },
            { key: 'created', label: '创建时间', value: formatAdminDateTime(row.created_at) },
            { key: 'paid', label: '支付时间', value: formatAdminDateTime(row.paid_at) },
          ]"
        />
      </div>
    </div>

    <el-pagination
      v-if="(orders.meta.value?.total ?? 0) > 0"
      class="admin-pagination"
      :current-page="pages.page"
      :page-size="orders.meta.value?.page_size ?? 50"
      :total="orders.meta.value?.total ?? 0"
      layout="prev, pager, next, total"
      background
      hide-on-single-page
      @current-change="setPage"
    />
  </section>
</template>

<script setup lang="ts">
import {
  fetchAdminOrders,
  type AdminOrder,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminPagedResource, useAdminViewport } from '@/composables/useAdminResource'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice, queryText } from '@/utils/query'
import { adminStatusOptions, adminStatusValues, formatAdminDateTime, formatAdminMoney } from '@/utils/adminFormat'

const isMobile = useAdminViewport()

interface OrderFilters {
  [key: string]: string
  status: string
  search: string
}

const orderDefaults = { status: '', search: '' }
const statusOptions = adminStatusOptions('order')

const query = useAdminQueryState<OrderFilters>({
  defaults: orderDefaults,
  pageKeys: ['page'],
  decode: query => ({
    status: queryChoice(query, 'status', adminStatusValues('order'), orderDefaults.status),
    search: queryText(query, 'search'),
  }),
  encode: (filters, pages) => ({
    status: filters.status === orderDefaults.status ? undefined : filters.status,
    search: filters.search || undefined,
    page: pages.page === 1 ? undefined : pages.page,
  }),
  onApplied: (_filters, pages): void | Promise<void> => orders.load(pages.page),
})

const orders = useAdminPagedResource<AdminOrder>(async page => {
  const { status, search } = query.applied.value
  return await fetchAdminOrders({ status, search }, page)
})
const { draft, pages } = query
void query.initialize()

function reloadCurrentPage() {
  void orders.load(pages.value.page)
}

function submitOrderFilters() {
  void query.submit({ page: 1 })
}

function setPage(page: number) {
  void query.setPage('page', page)
}
</script>
