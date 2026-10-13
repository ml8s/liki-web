<template>
  <section>
    <div class="page-heading">
      <h1>财务</h1>
      <el-button @click="load">
        刷新
      </el-button>
    </div>
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
            @click="loadReport"
          >
            刷新
          </el-button>
        </div>
      </template>
      <el-form inline>
        <el-form-item label="周期">
          <el-select
            v-model="reportDays"
            style="width: 120px"
            @change="loadReport"
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
      <el-table
        v-loading="reportLoading"
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

    <el-card
      shadow="never"
      class="report-card"
    >
      <template #header>
        <div class="card-heading">
          财务任务
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="loadTasks"
      >
        <el-form-item label="状态">
          <el-select
            v-model="status"
            style="width: 140px"
          >
            <el-option
              label="待处理"
              value="pending"
            />
            <el-option
              label="已追回"
              value="resolved"
            />
            <el-option
              label="已豁免"
              value="rejected"
            />
            <el-option
              label="全部"
              value=""
            />
          </el-select>
        </el-form-item>
        <el-button
          type="primary"
          @click="loadTasks"
        >
          查询
        </el-button>
      </el-form>

      <el-table
        v-loading="loading"
        :data="tasks"
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
            <el-tag
              :type="adminStatusTagType(row.status)"
              size="small"
            >
              {{ row.status }}
            </el-tag>
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
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  fetchAdminFinanceReport, fetchAdminFinanceTasks, resolveAdminFinanceTask,
  type AdminFinanceReport, type AdminFinanceTask,
} from '@/api/admin'
import { useAdminStore } from '@/stores/admin'
import { adminStatusTagType, financeTaskTypeLabel, formatAdminMoney } from '@/utils/adminFormat'
import { formatDateTime } from '@/utils/format'

const admin = useAdminStore()
const loading = ref(false)
const status = ref('pending')
const tasks = ref<AdminFinanceTask[]>([])
const reportDays = ref<7 | 30 | 90>(30)
const reportLoading = ref(false)
const report = ref<AdminFinanceReport | null>(null)

const reportRows = computed(() => [
  { label: '一次性收入', values: report.value?.one_time_paid ?? [] },
  { label: '订阅收入', values: report.value?.subscription_paid ?? [] },
  { label: '订阅退款', values: report.value?.subscription_refunded ?? [] },
  { label: '产品净收入', values: report.value?.net_product_revenue ?? [] },
  { label: '合作佣金净额', values: report.value?.referral_commission_net ?? [] },
  { label: '已付提现', values: report.value?.referral_payout_paid ?? [] },
  { label: '当前待处理提现', values: report.value?.referral_pending_payout ?? [] },
])

onMounted(load)

async function load() {
  await Promise.all([loadTasks(), loadReport()])
}

async function loadTasks() {
  loading.value = true
  try {
    tasks.value = (await fetchAdminFinanceTasks(status.value)).tasks ?? []
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '查询财务任务失败')
  } finally {
    loading.value = false
  }
}

async function loadReport() {
  reportLoading.value = true
  try {
    report.value = await fetchAdminFinanceReport(reportDays.value)
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '查询财务汇总失败')
  } finally {
    reportLoading.value = false
  }
}

async function resolve(row: AdminFinanceTask, result: 'resolved' | 'rejected') {
  const title = result === 'resolved' ? '追回佣金' : '豁免佣金'
  const { value } = await ElMessageBox.prompt('请输入处理说明。', title, {
    confirmButtonText: '提交',
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
.page-heading { display: flex; justify-content: space-between; align-items: center; }
.report-card { margin-bottom: 18px; }
.card-heading { display: flex; justify-content: space-between; align-items: center; }
.money-tag { margin: 0 4px 4px 0; }
h1 { margin: 0 0 16px; font-size: 24px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.muted { color: #6b7280; font-size: 12px; }
</style>
