<template>
  <section>
    <AdminPageHeader
      title="用户权益"
      description="按邮箱查询支持工单中的访问权限，并保留操作审计。"
    />

    <el-card
      v-if="admin.can('entitlement.manage')"
      shadow="never"
      class="compensation-card"
    >
      <template #header>
        客服补偿
      </template>
      <el-alert
        v-if="compensationError"
        :title="compensationError"
        type="error"
        show-icon
        :closable="false"
        class="query-error"
      />
      <el-form
        ref="compensationFormRef"
        :model="compensation"
        :rules="compensationRules"
        inline
        @submit.prevent="compensate"
      >
        <el-form-item
          label="用户邮箱"
          prop="email"
        >
          <el-input
            v-model="compensation.email"
            type="email"
            name="compensation-email"
            autocomplete="off"
            spellcheck="false"
            placeholder="user@example.com"
          />
        </el-form-item>
        <el-form-item
          label="模板"
          prop="template"
        >
          <el-select
            v-model="compensation.template"
            class="admin-filter-control"
          >
            <el-option
              label="补偿 7 天"
              value="support_7d"
            />
            <el-option
              label="补偿 30 天"
              value="support_30d"
            />
          </el-select>
        </el-form-item>
        <el-form-item
          label="原因"
          prop="reason"
        >
          <el-input
            v-model="compensation.reason"
            name="compensation-reason"
            autocomplete="off"
            placeholder="服务事故 / 工单编号"
          />
        </el-form-item>
        <el-button
          type="warning"
          plain
          :loading="compensating"
          native-type="submit"
        >
          发放补偿
        </el-button>
      </el-form>
    </el-card>

    <el-form
      class="filters"
      inline
      @submit.prevent="submitEntitlementFilters"
    >
      <el-form-item label="用户邮箱">
        <el-input
          v-model="email"
          type="email"
          name="entitlement-email"
          autocomplete="off"
          spellcheck="false"
          placeholder="user@example.com"
          clearable
        />
      </el-form-item>
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
        :loading="loading"
        native-type="submit"
      >
        查询
      </el-button>
    </el-form>

    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />

    <el-empty
      v-else-if="!loading && !entitlements.length"
      :description="queried ? '未找到用户权益' : '请输入邮箱后查询'"
    />
    <div
      v-else
      v-loading="loading"
      :aria-busy="loading"
    >
      <el-table
        v-if="!isMobile"
        :data="entitlements"
        row-key="id"
      >
        <el-table-column
          prop="email"
          label="邮箱"
          min-width="190"
          show-overflow-tooltip
        />
        <el-table-column
          prop="product_id"
          label="产品"
          width="120"
        />
        <el-table-column
          label="技能"
          min-width="120"
        >
          <template #default="{ row }">
            <el-tag
              v-for="skill in row.skills"
              :key="skill"
              size="small"
              class="skill-tag"
            >
              {{ skill }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="状态"
          width="100"
        >
          <template #default="{ row }">
            <AdminStatusTag
              domain="entitlement"
              :status="row.status"
            />
          </template>
        </el-table-column>
        <el-table-column
          label="有效期"
          min-width="300"
        >
          <template #default="{ row }">
            {{ formatAdminDateTime(row.starts_at) }} — {{ formatAdminDateTime(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column
          label="来源"
          min-width="170"
        >
          <template #default="{ row }">
            <div>{{ row.source }}</div>
            <div class="mono muted">
              {{ row.source_id }}
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="admin.can('entitlement.manage')"
          label="操作"
          width="170"
          fixed="right"
        >
          <template #default="{ row }">
            <el-button
              size="small"
              :disabled="row.status === 'revoked'"
              @click="openExpire(row as AdminEntitlement)"
            >
              改期
            </el-button>
            <el-button
              size="small"
              type="danger"
              plain
              :disabled="row.status === 'revoked'"
              @click="revoke(row as AdminEntitlement)"
            >
              撤销
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="isMobile">
        <AdminRecordCard
          v-for="row in entitlements"
          :key="row.id"
          status-domain="entitlement"
          :fields="[
            { key: 'email', label: '邮箱', value: row.email },
            { key: 'product_id', label: '产品', value: row.product_id },
            { key: 'skills', label: '技能', value: row.skills.join(', ') },
            { key: 'status', label: '状态', value: row.status, kind: 'status' },
            { key: 'expires', label: '有效期', value: `${formatAdminDateTime(row.starts_at)} — ${formatAdminDateTime(row.expires_at)}` },
            { key: 'source', label: '来源', value: row.source },
            { key: 'source_id', label: '来源 ID', value: row.source_id, kind: 'mono' },
          ]"
        >
          <template
            v-if="admin.can('entitlement.manage')"
            #actions
          >
            <el-button
              size="small"
              :disabled="row.status === 'revoked'"
              @click="openExpire(row)"
            >
              改期
            </el-button>
            <el-button
              size="small"
              type="danger"
              plain
              :disabled="row.status === 'revoked'"
              @click="revoke(row)"
            >
              撤销
            </el-button>
          </template>
        </AdminRecordCard>
      </div>
    </div>

    <el-dialog
      v-model="expireDialog"
      title="修改权益到期时间"
      width="min(430px, calc(100vw - 32px))"
    >
      <el-alert
        v-if="expireError"
        :title="expireError"
        type="error"
        show-icon
        :closable="false"
        class="query-error"
      />
      <el-form
        ref="expireFormRef"
        :model="expireForm"
        :rules="expireRules"
        label-position="top"
      >
        <el-form-item
          label="到期时间"
          prop="expiresAt"
        >
          <el-date-picker
            v-model="expireForm.expiresAt"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            format="YYYY-MM-DD HH:mm"
            class="admin-fluid-control"
          />
        </el-form-item>
        <el-form-item
          label="操作原因"
          prop="reason"
        >
          <el-input
            v-model="expireForm.reason"
            name="expire-reason"
            autocomplete="off"
            type="textarea"
            :rows="3"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="expireDialog = false">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="saveExpiry"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  compensateAdminEntitlement, expireAdminEntitlement, fetchAdminEntitlements, revokeAdminEntitlement,
  type AdminEntitlement,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { adminStatusOptions, adminStatusValues, formatAdminDateTime, toDatetimeLocal } from '@/utils/adminFormat'
import { useAdminViewport } from '@/composables/useAdminResource'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice } from '@/utils/query'

const isMobile = useAdminViewport()

const admin = useAdminStore()

interface EntitlementFilters {
  [key: string]: string
  status: string
}

// email 是敏感值：只保留在本地输入框，不进 URL；status 走统一查询状态契约。
const entitlementDefaults = { status: '' }
const email = ref('')
const statusOptions = adminStatusOptions('entitlement')
const loading = ref(false)
const queried = ref(false)
const saving = ref(false)
const error = ref('')
const entitlements = ref<AdminEntitlement[]>([])
const expireDialog = ref(false)
const editing = ref<AdminEntitlement | null>(null)
const expireForm = ref({ expiresAt: '', reason: '' })
const compensationFormRef = ref<FormInstance>()
const expireFormRef = ref<FormInstance>()
const compensationError = ref('')
const expireError = ref('')
const compensating = ref(false)
const compensationRules: FormRules = {
  email: [
    { required: true, whitespace: true, message: '请填写用户邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  reason: [
    { required: true, whitespace: true, message: '请填写原因或工单编号', trigger: 'blur' },
  ],
}
const expireRules: FormRules = {
  expiresAt: [{ required: true, message: '请选择到期时间', trigger: 'change' }],
  reason: [{ required: true, whitespace: true, message: '请填写操作原因', trigger: 'blur' }],
}
const compensation = ref({
  email: '',
  template: 'support_7d' as 'support_7d' | 'support_30d',
  reason: '',
  idempotency_key: '',
})

// The idempotency key represents one intended operation. Changing the intended
// operation starts a new key; resubmitting an unchanged form preserves it.
watch(() => [compensation.value.email, compensation.value.template, compensation.value.reason], () => {
  compensation.value.idempotency_key = ''
})

const query = useAdminQueryState<EntitlementFilters>({
  defaults: entitlementDefaults,
  pageKeys: [],
  decode: query => ({
    status: queryChoice(query, 'status', adminStatusValues('entitlement'), entitlementDefaults.status),
  }),
  encode: filters => ({
    status: filters.status === entitlementDefaults.status ? undefined : filters.status,
  }),
  onApplied: (): Promise<void> => load(),
})

const { draft } = query
void query.initialize()

function submitEntitlementFilters() {
  void query.submit()
}

async function load() {
  if (!email.value.trim()) {
    error.value = ''
    entitlements.value = []
    queried.value = false
    return
  }
  loading.value = true
  error.value = ''
  queried.value = true
  try {
    const payload = await fetchAdminEntitlements(email.value.trim(), query.applied.value.status)
    entitlements.value = payload.entitlements ?? []
  } catch (requestError) {
    error.value = requestError instanceof Error ? requestError.message : '查询失败'
    entitlements.value = []
  } finally {
    loading.value = false
  }
}

function openExpire(row: AdminEntitlement) {
  editing.value = row
  expireForm.value = { expiresAt: toDatetimeLocal(row.expires_at), reason: '' }
  expireDialog.value = true
}

async function saveExpiry() {
  if (!editing.value || !expireFormRef.value) return
  expireError.value = ''
  try {
    await expireFormRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await expireAdminEntitlement(editing.value.id, new Date(expireForm.value.expiresAt).toISOString(), expireForm.value.reason.trim())
    ElMessage.success('权益已更新')
    expireDialog.value = false
    await load()
  } catch (requestError) {
    expireError.value = requestError instanceof Error ? requestError.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function revoke(row: AdminEntitlement) {
  try {
    const { value } = await ElMessageBox.prompt('请输入撤销原因，该操作会写入审计。', '撤销权益', {
      confirmButtonText: '撤销',
      cancelButtonText: '取消',
      inputValidator: (input: string) => Boolean(input.trim()) || '原因必填',
    })
    await revokeAdminEntitlement(row.id, value.trim())
    ElMessage.success('权益已撤销')
    await load()
  } catch {
    // User cancelled the irreversible-action confirmation.
  }
}

async function compensate() {
  if (!compensationFormRef.value) return
  compensationError.value = ''
  try {
    await compensationFormRef.value.validate()
  } catch {
    return
  }
  const request = {
    email: compensation.value.email.trim(),
    template: compensation.value.template,
    reason: compensation.value.reason.trim(),
    idempotency_key: compensation.value.idempotency_key,
  }
  if (!request.idempotency_key) {
    request.idempotency_key = crypto.randomUUID()
    compensation.value.idempotency_key = request.idempotency_key
  }
  compensating.value = true
  try {
    const result = await compensateAdminEntitlement(
      request.email, request.template, request.reason, request.idempotency_key,
    )
    if (result.email_queued) {
      ElMessage.success('补偿已发放，通知邮件已排队')
    } else {
      ElMessage.warning('补偿已发放，但通知邮件排队失败；请记录工单并联系运维。')
    }
    email.value = request.email
    compensation.value.email = request.email
    compensation.value.reason = ''
    compensation.value.idempotency_key = ''
    await load()
  } catch (requestError) {
    compensationError.value = requestError instanceof Error ? requestError.message : '补偿失败'
  } finally {
    compensating.value = false
  }
}
</script>

<style scoped>
.compensation-card { margin-bottom: var(--space-4); }
.skill-tag + .skill-tag { margin-left: var(--space-1); }
</style>
