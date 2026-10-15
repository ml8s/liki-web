<template>
  <section>
    <AdminPageHeader
      title="管理员"
      description="能力精确授权；admin.super 会获得全部能力。"
    />

    <el-result
      v-if="!admin.can('admin.user.manage')"
      icon="warning"
      title="无管理员管理能力"
    />
    <template v-else>
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        show-icon
        :closable="false"
      />

      <el-card
        shadow="never"
        class="toolbar-card"
      >
        <el-form
          inline
          @submit.prevent="createAdmin"
        >
          <el-form-item label="管理员邮箱">
            <el-input
              v-model="newEmail"
              type="email"
              name="admin-email"
              autocomplete="off"
              spellcheck="false"
              placeholder="ops@example.com"
            />
          </el-form-item>
          <el-button
            type="primary"
            :loading="creating"
            @click="createAdmin"
          >
            新增
          </el-button>
        </el-form>
      </el-card>

      <div
        v-loading="loading"
        :aria-busy="loading"
      >
        <el-table
          v-if="!isMobile"
          :data="accounts"
          row-key="email"
        >
          <el-table-column
            prop="email"
            label="邮箱"
            min-width="210"
          />
          <el-table-column
            label="状态"
            width="100"
          >
            <template #default="{ row }">
              <AdminStatusTag :status="row.status" />
            </template>
          </el-table-column>
          <el-table-column
            label="能力"
            min-width="320"
          >
            <template #default="{ row }">
              <el-tag
                v-for="item in activeCapabilities(row as AdminAccount)"
                :key="item.capability"
                size="small"
                class="cap-tag"
                closable
                @close="revokeCapability(row as AdminAccount, item.capability)"
              >
                {{ capabilityLabel(item.capability) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            label="授权"
            width="220"
          >
            <template #default="{ row }">
              <el-select
                :model-value="grantCapability"
                aria-label="待授予能力"
                placeholder="选择能力"
                size="small"
                @change="(value: string) => grantCapability = value as AdminCapability"
              >
                <el-option
                  v-for="capability in capabilityOptions"
                  :key="capability"
                  :label="capabilityLabel(capability)"
                  :value="capability"
                />
              </el-select>
              <el-button
                class="grant-btn"
                size="small"
                type="primary"
                plain
                @click="grant(row as AdminAccount)"
              >
                授权
              </el-button>
            </template>
          </el-table-column>
          <el-table-column
            label="账号"
            width="150"
            fixed="right"
          >
            <template #default="{ row }">
              <el-button
                size="small"
                :type="row.status === 'active' ? 'danger' : 'primary'"
                plain
                @click="toggleStatus(row as AdminAccount)"
              >
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div
          v-if="isMobile && !loading && !accounts.length"
        >
          <el-empty description="暂无管理员" />
        </div>
        <div v-if="isMobile">
          <AdminRecordCard
            v-for="row in accounts"
            :key="row.email"
            :fields="[
              { key: 'email', label: '邮箱', value: row.email },
              { key: 'status', label: '状态', value: row.status, kind: 'status' },
              { key: 'capabilities', label: '能力', value: activeCapabilities(row).map(item => capabilityLabel(item.capability)).join(', ') },
            ]"
          >
            <template #field="{ field }">
              <div
                v-if="field.key === 'capabilities' && row"
                class="admin-capability-values"
              >
                <el-tag
                  v-for="item in activeCapabilities(row)"
                  :key="item.capability"
                  size="small"
                  class="cap-tag"
                  closable
                  @close="revokeCapability(row, item.capability)"
                >
                  {{ capabilityLabel(item.capability) }}
                </el-tag>
                <span
                  v-if="!activeCapabilities(row).length"
                  class="muted"
                >-</span>
              </div>
              <span v-else>{{ field.value || '-' }}</span>
            </template>
            <template #actions>
              <el-select
                :model-value="grantCapability"
                aria-label="待授予能力"
                placeholder="选择能力"
                size="small"
                @change="(value: string) => grantCapability = value as AdminCapability"
              >
                <el-option
                  v-for="capability in capabilityOptions"
                  :key="capability"
                  :label="capabilityLabel(capability)"
                  :value="capability"
                />
              </el-select>
              <el-button
                size="small"
                type="primary"
                plain
                @click="grant(row)"
              >
                授权
              </el-button>
              <el-button
                size="small"
                :type="row.status === 'active' ? 'danger' : 'primary'"
                plain
                @click="toggleStatus(row)"
              >
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </el-button>
            </template>
          </AdminRecordCard>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createAdminAccount, fetchAdminAccounts, grantAdminCapability, revokeAdminCapability,
  setAdminAccountStatus, type AdminAccount, type AdminCapability,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { capabilityLabel } from '@/utils/adminFormat'
import { useAdminViewport } from '@/composables/useAdminResource'

const isMobile = useAdminViewport()

const admin = useAdminStore()
const accounts = ref<AdminAccount[]>([])
const loading = ref(false)
const creating = ref(false)
const error = ref('')
const newEmail = ref('')
const grantCapability = ref<AdminCapability | ''>('')

const capabilityOptions: AdminCapability[] = [
  'entitlement.read', 'entitlement.manage', 'partner.read', 'partner.approve', 'partner.freeze',
  'partner.unfreeze', 'commission.read', 'commission.void', 'payout.read', 'payout.approve',
  'payout.reject', 'payout.mark_paid', 'audit.read', 'admin.user.manage', 'admin.super',
]

onMounted(load)

function activeCapabilities(row: AdminAccount) {
  return (row.capabilities ?? []).filter(item => item.status === 'active')
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    accounts.value = (await fetchAdminAccounts()).admins ?? []
  } catch (requestError) {
    error.value = requestError instanceof Error ? requestError.message : '查询失败'
  } finally {
    loading.value = false
  }
}

async function createAdmin() {
  if (!newEmail.value.trim()) return
  creating.value = true
  try {
    await createAdminAccount(newEmail.value.trim())
    ElMessage.success('管理员已新增')
    newEmail.value = ''
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '创建失败')
  } finally {
    creating.value = false
  }
}

async function requireReason(title: string) {
  const { value } = await ElMessageBox.prompt('请输入原因，该操作会写入审计。', title, {
    confirmButtonText: title,
    cancelButtonText: '取消',
    inputValidator: (input: string) => Boolean(input.trim()) || '原因必填',
  })
  return value.trim()
}

async function grant(row: AdminAccount) {
  if (!grantCapability.value) {
    ElMessage.warning('请选择能力')
    return
  }
  const reason = await requireReason('授予权限').catch(() => null)
  if (!reason) return
  try {
    await grantAdminCapability(row.email, grantCapability.value, reason)
    ElMessage.success('权限已授予')
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '授权失败')
  }
}

async function toggleStatus(row: AdminAccount) {
  const disable = row.status === 'active'
  const reason = await requireReason(disable ? '禁用管理员' : '启用管理员').catch(() => null)
  if (!reason) return
  try {
    await setAdminAccountStatus(row.email, disable ? 'disabled' : 'active', reason)
    ElMessage.success(disable ? '管理员已禁用' : '管理员已启用')
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '状态更新失败')
  }
}

// Active capability chips are deliberately clickable: revoke is privileged and
// must always pass through the audited reason prompt.
async function revokeCapability(account: AdminAccount, capability: AdminCapability) {
  const reason = await requireReason(`撤销 ${capabilityLabel(capability)}`).catch(() => null)
  if (!reason) return
  try {
    await revokeAdminCapability(account.email, capability, reason)
    ElMessage.success('权限已撤销')
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '撤销失败')
  }
}
</script>

<style scoped>
.toolbar-card { margin-bottom: 16px; }
.cap-tag { margin: 0 4px 4px 0; cursor: pointer; }
.grant-btn { margin-left: 6px; }
</style>
