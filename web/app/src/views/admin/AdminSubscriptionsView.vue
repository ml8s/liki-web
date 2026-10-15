<template>
  <section>
    <AdminPageHeader title="订阅">
      <template #actions>
        <el-button
          :loading="subscriptions.loading.value"
          @click="reloadCurrentPage"
        >
          刷新
        </el-button>
      </template>
    </AdminPageHeader>

    <el-form
      inline
      @submit.prevent="submitSubscriptionFilters"
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
          placeholder="订阅号 / 邮箱 / 服务商订阅号"
          clearable
          spellcheck="false"
          class="admin-search-control"
        />
      </el-form-item>
      <el-button
        type="primary"
        @click="submitSubscriptionFilters"
      >
        查询
      </el-button>
    </el-form>

    <el-alert
      v-if="subscriptions.error.value"
      :title="subscriptions.error.value"
      type="error"
      show-icon
      :closable="false"
      class="query-error"
    />

    <div
      v-loading="subscriptions.loading.value"
      :aria-busy="subscriptions.loading.value"
    >
      <el-table
        v-if="!isMobile"
        :data="subscriptions.data.value"
        row-key="id"
      >
        <el-table-column
          prop="id"
          label="订阅号"
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
          label="状态"
          width="110"
        >
          <template #default="{ row }">
            <AdminStatusTag
              domain="subscription"
              :status="row.status"
            />
          </template>
        </el-table-column>
        <el-table-column
          label="服务商订阅号"
          min-width="190"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span class="mono muted">{{ row.provider_subscription_id || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="当前周期结束"
          width="180"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.current_period_end) }}
          </template>
        </el-table-column>
        <el-table-column
          label="下次扣费"
          width="180"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.next_billing_at) }}
          </template>
        </el-table-column>
        <el-table-column
          label="期末取消"
          width="110"
        >
          <template #default="{ row }">
            {{ row.cancel_at_period_end ? '是' : '否' }}
          </template>
        </el-table-column>
        <el-table-column
          label="创建时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>

      <div
        v-if="isMobile && !subscriptions.loading.value && !subscriptions.data.value.length"
      >
        <el-empty description="暂无订阅" />
      </div>
      <div v-if="isMobile">
        <AdminRecordCard
          v-for="row in subscriptions.data.value"
          :key="row.id"
          status-domain="subscription"
          :fields="[
            { key: 'id', label: '订阅号', value: row.id, kind: 'mono' },
            { key: 'email', label: '用户邮箱', value: row.email },
            { key: 'product', label: '产品', value: row.product_id },
            { key: 'status', label: '状态', value: row.status, kind: 'status' },
            { key: 'provider', label: '服务商订阅号', value: row.provider_subscription_id, kind: 'mono' },
            { key: 'periodEnd', label: '当前周期结束', value: formatDateTime(row.current_period_end) },
            { key: 'nextBilling', label: '下次扣费', value: formatDateTime(row.next_billing_at) },
            { key: 'cancelAtPeriodEnd', label: '期末取消', value: row.cancel_at_period_end ? '是' : '否' },
            { key: 'created', label: '创建时间', value: formatDateTime(row.created_at) },
          ]"
        />
      </div>
    </div>

    <el-pagination
      v-if="(subscriptions.meta.value?.total ?? 0) > 0"
      class="admin-pagination"
      :current-page="pages.page"
      :page-size="subscriptions.meta.value?.page_size ?? 50"
      :total="subscriptions.meta.value?.total ?? 0"
      layout="prev, pager, next, total"
      background
      hide-on-single-page
      @current-change="setPage"
    />
  </section>
</template>

<script setup lang="ts">
import {
  fetchAdminSubscriptions,
  type AdminSubscription,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminPagedResource, useAdminViewport } from '@/composables/useAdminResource'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice, queryText } from '@/utils/query'
import { adminStatusOptions } from '@/utils/adminFormat'
import { formatDateTime } from '@/utils/format'

const isMobile = useAdminViewport()

interface SubscriptionFilters {
  [key: string]: string
  status: string
  search: string
}

const subscriptionDefaults = { status: '', search: '' }
const statusOptions = adminStatusOptions('subscription')

const query = useAdminQueryState<SubscriptionFilters>({
  defaults: subscriptionDefaults,
  pageKeys: ['page'],
  decode: query => ({
    status: queryChoice(query, 'status', statusOptions.map(option => option.value), subscriptionDefaults.status),
    search: queryText(query, 'search'),
  }),
  encode: (filters, pages) => ({
    status: filters.status === subscriptionDefaults.status ? undefined : filters.status,
    search: filters.search || undefined,
    page: pages.page === 1 ? undefined : pages.page,
  }),
  onApplied: (_filters, pages): void | Promise<void> => subscriptions.load(pages.page),
})

const subscriptions = useAdminPagedResource<AdminSubscription>(async page => {
  const { status, search } = query.applied.value
  return await fetchAdminSubscriptions({ status, search }, page)
})
const { draft, pages } = query
void query.initialize()

function reloadCurrentPage() {
  void subscriptions.load(pages.value.page)
}

function submitSubscriptionFilters() {
  void query.submit({ page: 1 })
}

function setPage(page: number) {
  void query.setPage('page', page)
}
</script>
