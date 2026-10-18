<template>
  <section>
    <AdminPageHeader title="提现复核">
      <template #actions>
        <el-button
          :loading="payouts.loading.value"
          @click="reloadCurrentPage"
        >
          刷新
        </el-button>
      </template>
    </AdminPageHeader>
    <el-form
      inline
      @submit.prevent="submitPayoutFilters"
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
      <el-button
        type="primary"
        @click="submitPayoutFilters"
      >
        查询
      </el-button>
    </el-form>

    <el-alert
      v-if="payouts.error.value"
      :title="payouts.error.value"
      type="error"
      show-icon
      :closable="false"
      class="query-error"
    />
    <div
      v-loading="payouts.loading.value"
      :aria-busy="payouts.loading.value"
    >
      <el-table
        v-if="!isMobile"
        :data="payouts.data.value"
        row-key="id"
      >
        <el-table-column
          label="金额"
          width="130"
        >
          <template #default="{ row }">
            {{ formatAdminMoney(row.amount_minor, row.currency) }}
          </template>
        </el-table-column>
        <el-table-column
          prop="referrer_code"
          label="推广码"
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column
          prop="referrer_id"
          label="合作伙伴 ID"
          min-width="180"
          show-overflow-tooltip
        />
        <el-table-column
          label="角色"
          width="90"
        >
          <template #default="{ row }">
            {{ adminRoleLabel(row.role) }}
          </template>
        </el-table-column>
        <el-table-column
          label="方式 / 账号"
          min-width="210"
        >
          <template #default="{ row }">
            <div>{{ adminPaymentMethodLabel(row.method) }}</div>
            <div class="mono muted">
              {{ revealed === row.id ? row.account : maskAdminAccount(row.account) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="状态"
          width="110"
        >
          <template #default="{ row }">
            <AdminStatusTag
              domain="payout"
              :status="row.status"
            />
          </template>
        </el-table-column>
        <el-table-column
          label="时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatAdminDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column
          label="操作"
          width="240"
          fixed="right"
        >
          <template #default="{ row }">
            <template v-if="actionableStatuses.includes(row.status)">
              <el-button
                v-if="admin.can('payout.mark_paid')"
                size="small"
                @click="reveal(row as AdminPayout)"
              >
                显示账号
              </el-button>
              <el-button
                v-if="admin.can('payout.approve') && row.status === 'requested'"
                size="small"
                type="primary"
                @click="act(row as AdminPayout, 'approve')"
              >
                批准
              </el-button>
              <el-button
                v-if="admin.can('payout.reject')"
                size="small"
                type="danger"
                plain
                @click="act(row as AdminPayout, 'reject')"
              >
                拒绝
              </el-button>
              <el-button
                v-if="admin.can('payout.mark_paid') && row.status === 'approved'"
                size="small"
                type="success"
                @click="act(row as AdminPayout, 'mark-paid')"
              >
                标记打款
              </el-button>
            </template>
            <span
              v-else
              class="muted"
            >{{ payoutActionPlaceholder(row.status) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div
        v-if="isMobile && !payouts.loading.value && !payouts.data.value.length"
      >
        <el-empty description="暂无提现记录" />
      </div>
      <div v-if="isMobile">
        <AdminRecordCard
          v-for="row in payouts.data.value"
          :key="row.id"
          status-domain="payout"
          :fields="[
            { key: 'amount', label: '金额', value: row.amount_minor, kind: 'money', currency: row.currency },
            { key: 'code', label: '推广码', value: row.referrer_code },
            { key: 'partner', label: '合作伙伴 ID', value: row.referrer_id, kind: 'mono' },
            { key: 'role', label: '角色', value: adminRoleLabel(row.role) },
            { key: 'method', label: '方式', value: adminPaymentMethodLabel(row.method) },
            { key: 'account', label: '账号', value: revealed === row.id ? row.account : maskAdminAccount(row.account), kind: 'mono' },
            { key: 'status', label: '状态', value: row.status, kind: 'status' },
            { key: 'created', label: '时间', value: formatAdminDateTime(row.created_at) },
          ]"
        >
          <template #actions>
            <template v-if="actionableStatuses.includes(row.status)">
              <el-button
                v-if="admin.can('payout.mark_paid')"
                size="small"
                @click="reveal(row)"
              >
                显示账号
              </el-button>
              <el-button
                v-if="admin.can('payout.approve') && row.status === 'requested'"
                size="small"
                type="primary"
                @click="act(row, 'approve')"
              >
                批准
              </el-button>
              <el-button
                v-if="admin.can('payout.reject')"
                size="small"
                type="danger"
                plain
                @click="act(row, 'reject')"
              >
                拒绝
              </el-button>
              <el-button
                v-if="admin.can('payout.mark_paid') && row.status === 'approved'"
                size="small"
                type="success"
                @click="act(row, 'mark-paid')"
              >
                标记打款
              </el-button>
            </template>
            <span
              v-else
              class="muted"
            >{{ payoutActionPlaceholder(row.status) }}</span>
          </template>
        </AdminRecordCard>
      </div>
    </div>

    <el-pagination
      v-if="(payouts.meta.value?.total ?? 0) > 0"
      class="admin-pagination"
      :current-page="pages.page"
      :page-size="payouts.meta.value?.page_size ?? 100"
      :total="payouts.meta.value?.total ?? 0"
      layout="prev, pager, next, total"
      background
      hide-on-single-page
      @current-change="setPage"
    />
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  actOnAdminPayout, fetchAdminPayouts, revealAdminPayoutAccount, type AdminPayout,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { useAdminPagedResource, useAdminViewport } from '@/composables/useAdminResource'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice } from '@/utils/query'
import {
  adminPaymentMethodLabel, adminRoleLabel, adminStatusOptions, adminStatusValues, formatAdminDateTime,
  formatAdminMoney, maskAdminAccount,
} from '@/utils/adminFormat'

const isMobile = useAdminViewport()

const admin = useAdminStore()

interface PayoutFilters {
  [key: string]: string
  status: string
}

const payoutDefaults = { status: 'requested' }
const statusOptions = adminStatusOptions('payout')
const revealed = ref('')

// 仅 requested/approved 有可执行动作；终态行给占位避免操作列空白。
const actionableStatuses = ['requested', 'approved']
const terminalStatuses = ['paid', 'rejected', 'failed']

function payoutActionPlaceholder(status: string) {
  return terminalStatuses.includes(status) ? '已处理' : '-'
}

const query = useAdminQueryState<PayoutFilters>({
  defaults: payoutDefaults,
  pageKeys: ['page'],
  decode: query => ({
    status: queryChoice(query, 'status', adminStatusValues('payout'), payoutDefaults.status),
  }),
  encode: (filters, pages) => ({
    status: filters.status === payoutDefaults.status ? undefined : filters.status,
    page: pages.page === 1 ? undefined : pages.page,
  }),
  onApplied: (_filters, pages): void | Promise<void> => payouts.load(pages.page),
})

const payouts = useAdminPagedResource<AdminPayout>(async page => {
  const { status } = query.applied.value
  return await fetchAdminPayouts(status, page)
})
const { draft, pages } = query
void query.initialize()

function reloadCurrentPage() {
  void payouts.load(pages.value.page)
}

function submitPayoutFilters() {
  void query.submit({ page: 1 })
}

function setPage(page: number) {
  void query.setPage('page', page)
}

async function requireInput(title: string, label: string) {
  const { value } = await ElMessageBox.prompt(label, title, {
    confirmButtonText: title,
    cancelButtonText: '取消',
    inputValidator: (input: string) => Boolean(input.trim()) || '必填',
  })
  return value.trim()
}

async function reveal(row: AdminPayout) {
  const reason = await requireInput('显示收款账号', '请输入原因，该操作会写入管理记录。').catch(() => null)
  if (!reason) return
  try {
    const result = await revealAdminPayoutAccount(row.id, reason)
    row.account = result.account
    revealed.value = row.id
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '显示失败')
  }
}

async function act(row: AdminPayout, action: 'approve' | 'reject' | 'mark-paid') {
  let body: Record<string, string>
  if (action === 'approve' || action === 'reject') {
    const reason = await requireInput(action === 'approve' ? '批准提现' : '拒绝提现', '请输入原因。').catch(() => null)
    if (!reason) return
    body = { reason }
  } else {
    const reference = await requireInput('标记已打款', '请输入打款凭证号。').catch(() => null)
    if (!reference) return
    body = { reason: 'manual payout', payment_reference: reference }
  }
  try {
    await actOnAdminPayout(row.id, action, body)
    ElMessage.success('提现已更新')
    await payouts.load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '操作失败')
  }
}
</script>
