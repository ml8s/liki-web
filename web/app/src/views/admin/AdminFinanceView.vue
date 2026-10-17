<template>
  <section>
    <AdminPageHeader title="财务">
      <template #actions>
        <el-button @click="reloadTaskPage">
          刷新
        </el-button>
      </template>
    </AdminPageHeader>
    <el-card
      shadow="never"
      class="report-card"
    >
      <template #header>
        <div class="card-heading">
          经营汇总
          <el-button
            size="small"
            :loading="reportLoading"
            @click="loadAppliedReport"
          >
            刷新
          </el-button>
        </div>
      </template>
      <el-form inline>
        <el-form-item label="周期">
          <el-select
            v-model="draft.days"
            class="admin-filter-control"
            @change="submitFinanceFilters"
          >
            <el-option
              :value="7"
              label="近 7 天"
            />
            <el-option
              :value="30"
              label="近 30 天"
            />
            <el-option
              :value="90"
              label="近 90 天"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <el-alert
        v-if="reportError"
        :title="reportError"
        type="error"
        show-icon
        :closable="false"
        class="query-error"
      />
      <el-table
        v-loading="reportLoading"
        :aria-busy="reportLoading"
        :data="reportRows"
      >
        <el-table-column
          prop="label"
          label="指标"
          min-width="180"
        />
        <el-table-column
          label="金额"
          min-width="300"
        >
          <template #default="{ row }">
            <el-tag
              v-for="item in row.values"
              :key="item.currency"
              class="money-tag"
              size="small"
            >
              {{ formatAdminMoney(item.amount_minor, item.currency) }}
            </el-tag>
            <span
              v-if="!row.values.length"
              class="muted"
            >0</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="card-heading">
          财务任务
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="submitFinanceFilters"
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
          @click="reloadTaskPage"
        >
          查询
        </el-button>
      </el-form>
      <el-alert
        v-if="tasks.error.value"
        :title="tasks.error.value"
        type="error"
        show-icon
        :closable="false"
        class="query-error"
      />
      <div
        v-loading="tasks.loading.value"
        :aria-busy="tasks.loading.value"
      >
        <el-table
          v-if="!isMobile"
          :data="tasks.data.value"
          row-key="id"
        >
          <el-table-column
            label="类型 / 金额"
            min-width="220"
          >
            <template #default="{ row }">
              <div>{{ financeTaskTypeLabel(row.type) }}</div>
              <div class="mono muted">
                {{ formatAdminMoney(row.amount_minor, row.currency) }}
              </div>
            </template>
          </el-table-column>
          <el-table-column
            prop="beneficiary_id"
            label="受益人"
            min-width="180"
            show-overflow-tooltip
          />
          <el-table-column
            label="关联对象"
            min-width="220"
          >
            <template #default="{ row }">
              <div>{{ row.provider }}</div>
              <div class="mono muted">
                {{ row.provider_refund_id || row.payment_intent_id }}
              </div>
            </template>
          </el-table-column>
          <el-table-column
            label="状态"
            width="110"
          >
            <template #default="{ row }">
              <AdminStatusTag
                domain="finance_task"
                :status="row.status"
              />
            </template>
          </el-table-column>
          <el-table-column
            label="时间"
            width="180"
          >
            <template #default="{ row }">
              {{ formatDateTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column
            v-if="admin.can('commission.void')"
            label="操作"
            width="180"
            fixed="right"
          >
            <template #default="{ row }">
              <template v-if="row.status === 'pending'">
                <el-button
                  size="small"
                  type="primary"
                  @click="resolve(row as AdminFinanceTask, 'resolved')"
                >
                  追回
                </el-button>
                <el-button
                  size="small"
                  type="warning"
                  plain
                  @click="resolve(row as AdminFinanceTask, 'rejected')"
                >
                  豁免
                </el-button>
              </template>
              <span
                v-else
                class="muted"
              >已处理</span>
            </template>
          </el-table-column>
        </el-table>

        <div
          v-if="isMobile && !tasks.loading.value && !tasks.data.value.length"
        >
          <el-empty description="暂无财务任务" />
        </div>
        <div v-if="isMobile">
          <AdminRecordCard
            v-for="row in tasks.data.value"
            :key="row.id"
            status-domain="finance_task"
            :fields="[
              { key: 'type', label: '类型', value: financeTaskTypeLabel(row.type) },
              { key: 'amount', label: '金额', value: row.amount_minor, kind: 'money', currency: row.currency },
              { key: 'beneficiary', label: '受益人', value: row.beneficiary_id, kind: 'mono' },
              { key: 'provider', label: 'Provider', value: row.provider },
              { key: 'reference', label: '关联对象', value: row.provider_refund_id || row.payment_intent_id, kind: 'mono' },
              { key: 'status', label: '状态', value: row.status, kind: 'status' },
              { key: 'created', label: '时间', value: formatDateTime(row.created_at) },
            ]"
          >
            <template
              v-if="admin.can('commission.void') && row.status === 'pending'"
              #actions
            >
              <el-button
                size="small"
                type="primary"
                @click="resolve(row, 'resolved')"
              >
                追回
              </el-button>
              <el-button
                size="small"
                type="warning"
                plain
                @click="resolve(row, 'rejected')"
              >
                豁免
              </el-button>
            </template>
          </AdminRecordCard>
        </div>
      </div>

      <el-pagination
        v-if="(tasks.meta.value?.total ?? 0) > 0"
        class="admin-pagination"
        :current-page="pagesState.taskPage"
        :page-size="tasks.meta.value?.page_size ?? 100"
        :total="tasks.meta.value?.total ?? 0"
        layout="prev, pager, next, total"
        background
        hide-on-single-page
        @current-change="setTaskPage"
      />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  fetchAdminFinanceReport, fetchAdminFinanceTasks, resolveAdminFinanceTask,
  type AdminFinanceReport, type AdminFinanceTask,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { useAdminPagedResource, useAdminViewport } from '@/composables/useAdminResource'
import { adminStatusOptions, adminStatusValues, financeTaskTypeLabel, formatAdminMoney } from '@/utils/adminFormat'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice } from '@/utils/query'
import { formatDateTime } from '@/utils/format'

const isMobile = useAdminViewport()

const admin = useAdminStore()

interface FinanceFilters {
  [key: string]: string | number
  status: string
  days: 7 | 30 | 90
}

const tasks = useAdminPagedResource<AdminFinanceTask>(async page => {
  const { status } = query.applied.value
  return await fetchAdminFinanceTasks(status, page)
})
const reportLoading = ref(false)
const report = ref<AdminFinanceReport | null>(null)
const reportError = ref('')
const reportDays = computed(() => query.applied.value.days)

const reportRows = computed(() => [
  { label: '一次性收入', values: report.value?.one_time_paid ?? [] },
  { label: '订阅收入', values: report.value?.subscription_paid ?? [] },
  { label: '订阅退款', values: report.value?.subscription_refunded ?? [] },
  { label: '产品净收入', values: report.value?.net_product_revenue ?? [] },
  { label: '合作佣金净额', values: report.value?.partner_commission_net ?? [] },
  { label: '已付提现', values: report.value?.partner_payout_paid ?? [] },
  { label: '当前待处理提现', values: report.value?.partner_pending_payout ?? [] },
])

const statusOptions = adminStatusOptions('finance_task')

const query = useAdminQueryState<FinanceFilters>({
  defaults: { status: 'pending', days: 30 },
  pageKeys: ['taskPage'],
  decode: query => ({
    status: queryChoice(query, 'status', adminStatusValues('finance_task'), 'pending'),
    days: queryChoice(query, 'days', ['7', '30', '90'], '30') as unknown as 7 | 30 | 90,
  }),
  encode: (filters, pages) => ({
    status: filters.status === 'pending' ? undefined : filters.status,
    days: filters.days === 30 ? undefined : filters.days,
    taskPage: pages.taskPage === 1 ? undefined : pages.taskPage,
  }),
  onApplied: async (filters, pages): Promise<void> => {
    await Promise.all([
      tasks.load(pages.taskPage),
      loadReport(filters.days),
    ])
  },
})

const { draft } = query
const pagesState = query.pages
void query.initialize()

async function load() {
  await query.submit({ taskPage: 1 })
}

function reloadTaskPage() {
  void tasks.load(pagesState.value.taskPage)
}

function submitFinanceFilters() {
  void query.submit({ taskPage: 1 })
}

function setTaskPage(page: number) {
  void query.setPage('taskPage', page)
}

async function loadAppliedReport() {
  await loadReport(reportDays.value)
}

async function loadReport(days: 7 | 30 | 90 = reportDays.value) {
  reportLoading.value = true
  reportError.value = ''
  try {
    report.value = await fetchAdminFinanceReport(days)
  } catch (requestError) {
    reportError.value = requestError instanceof Error ? requestError.message : '查询财务汇总失败'
  } finally {
    reportLoading.value = false
  }
}

async function resolve(row: AdminFinanceTask, result: 'resolved' | 'rejected') {
  const title = result === 'resolved' ? '追回佣金' : '豁免佣金'
  const { value } = await ElMessageBox.prompt('请输入处理说明。', title, {
    confirmButtonText: title,
    cancelButtonText: '取消',
    inputValidator: (input: string) => Boolean(input.trim()) || '说明必填',
  }).catch(() => ({ value: null }))
  if (!value) return
  try {
    await resolveAdminFinanceTask(row.id, result, value.trim())
    ElMessage.success('财务任务已处理')
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '处理失败')
  }
}
</script>

<style scoped>
.report-card { margin-bottom: var(--space-5); }
.money-tag { margin: 0 4px 4px 0; }
</style>
