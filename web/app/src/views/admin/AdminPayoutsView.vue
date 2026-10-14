<template>
  <section>
    <div class="page-heading">
      <h1>提现复核</h1>
      <el-button
        :loading="payouts.loading.value"
        @click="loadPayouts"
      >
        刷新
      </el-button>
    </div>
    <el-form
      inline
      @submit.prevent="loadPayouts"
    >
      <el-form-item label="状态">
        <el-select
          v-model="status"
          style="width: 140px"
        >
          <el-option
            label="待审核"
            value="requested"
          />
          <el-option
            label="已批准"
            value="approved"
          />
          <el-option
            label="已拒绝"
            value="rejected"
          />
          <el-option
            label="已打款"
            value="paid"
          />
          <el-option
            label="失败"
            value="failed"
          />
          <el-option
            label="全部"
            value=""
          />
        </el-select>
      </el-form-item>
      <el-button
        type="primary"
        @click="loadPayouts"
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
    <el-table
      v-loading="payouts.loading.value"
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
        prop="role"
        label="角色"
        width="90"
      />
      <el-table-column
        label="方式 / 账号"
        min-width="210"
      >
        <template #default="{ row }">
          <div>{{ row.method }}</div>
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
          <el-tag
            :type="adminStatusTagType(row.status)"
            size="small"
          >
            {{ adminStatusLabel(row.status) }}
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
        label="操作"
        width="240"
        fixed="right"
      >
        <template #default="{ row }">
          <el-button
            v-if="admin.can('payout.mark_paid') && ['requested','approved'].includes(row.status)"
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
            v-if="admin.can('payout.reject') && ['requested','approved'].includes(row.status)"
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
      </el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  actOnAdminPayout, fetchAdminPayouts, revealAdminPayoutAccount, type AdminPayout,
} from '@/api/admin'
import { useAdminStore } from '@/stores/admin'
import { useRoute, useRouter } from 'vue-router'
import { useAdminResource } from '@/composables/useAdminResource'
import { queryChoice } from '@/utils/query'
import { adminStatusLabel, adminStatusTagType, formatAdminMoney, maskAdminAccount } from '@/utils/adminFormat'
import { formatDateTime } from '@/utils/format'

const admin = useAdminStore()
const route = useRoute()
const router = useRouter()
const status = ref(queryChoice(route.query, 'status', ['', 'requested', 'approved', 'rejected', 'paid', 'failed'], 'requested'))
const payouts = useAdminResource<AdminPayout>(async () => {
  const result = await fetchAdminPayouts(status.value)
  return result.payouts ?? []
})
const revealed = ref('')
const loadPayouts = payouts.load

onMounted(loadPayouts)

watch(status, async (value) => {
  await router.replace({ query: { ...route.query, status: value || undefined } })
})

async function requireInput(title: string, label: string) {
  const { value } = await ElMessageBox.prompt(label, title, {
    confirmButtonText: '确认',
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

<style scoped>
.page-heading { display: flex; justify-content: space-between; align-items: center; }
.query-error { margin-bottom: 12px; }
h1 { margin: 0 0 16px; font-size: 24px; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.muted { color: #6b7280; font-size: 12px; }
</style>
