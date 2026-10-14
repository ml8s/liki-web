<template>
  <section>
    <h1>合作伙伴</h1>
    <el-card
      shadow="never"
      class="block"
    >
      <template #header>
        <div class="card-heading">
          合作申请
          <el-button
            size="small"
            :loading="loading"
            @click="load"
          >
            刷新
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="loadApplications"
      >
        <el-form-item label="角色">
          <el-select
            v-model="applicationRole"
            style="width: 110px"
          >
            <el-option
              label="全部"
              value=""
            />
            <el-option
              label="KOL"
              value="kol"
            />
            <el-option
              label="Scout"
              value="scout"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="applicationStatus"
            style="width: 130px"
          >
            <el-option
              label="待审核"
              value="pending"
            />
            <el-option
              label="已通过"
              value="approved"
            />
            <el-option
              label="已拒绝"
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
          @click="loadApplications"
        >
          查询
        </el-button>
      </el-form>
      <el-table
        v-loading="loading"
        :data="applications"
        row-key="id"
      >
        <el-table-column
          prop="code"
          label="推广码"
          min-width="130"
        />
        <el-table-column
          prop="type"
          label="角色"
          width="90"
        />
        <el-table-column
          prop="note"
          label="说明"
          min-width="220"
          show-overflow-tooltip
        />
        <el-table-column
          label="状态"
          width="100"
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
          v-if="admin.can('partner.approve')"
          label="操作"
          width="160"
          fixed="right"
        >
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button
                size="small"
                type="primary"
                @click="decide(row as AdminApplication, true)"
              >
                通过
              </el-button>
              <el-button
                size="small"
                type="danger"
                plain
                @click="decide(row as AdminApplication, false)"
              >
                拒绝
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

    <el-card shadow="never">
      <template #header>
        <div class="card-heading">
          合作伙伴
          <el-button
            size="small"
            @click="loadPartners"
          >
            筛选
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="loadPartners"
      >
        <el-form-item label="角色">
          <el-select
            v-model="partnerRole"
            style="width: 110px"
          >
            <el-option
              label="全部"
              value=""
            />
            <el-option
              label="KOL"
              value="kol"
            />
            <el-option
              label="Scout"
              value="scout"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="partnerStatus"
            style="width: 120px"
          >
            <el-option
              label="全部"
              value=""
            />
            <el-option
              label="Active"
              value="active"
            />
            <el-option
              label="Frozen"
              value="frozen"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="搜索">
          <el-input
            v-model="partnerQuery"
            placeholder="ID / 推广码"
            clearable
          />
        </el-form-item>
      </el-form>
      <el-table
        v-loading="loading"
        :data="partners"
        row-key="user_id"
      >
        <el-table-column
          prop="code"
          label="推广码"
          min-width="130"
        />
        <el-table-column
          prop="user_id"
          label="User ID"
          min-width="220"
          show-overflow-tooltip
        />
        <el-table-column
          prop="role"
          label="角色"
          width="90"
        />
        <el-table-column
          label="状态"
          width="100"
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
          v-if="admin.canAny(['partner.freeze','partner.unfreeze'])"
          label="操作"
          width="110"
          fixed="right"
        >
          <template #default="{ row }">
            <el-button
              size="small"
              :type="row.status === 'active' ? 'danger' : 'primary'"
              plain
              :disabled="row.status === 'active' ? !admin.can('partner.freeze') : !admin.can('partner.unfreeze')"
              @click="togglePartner(row as AdminPartner)"
            >
              {{ row.status === 'active' ? '冻结' : '解冻' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  decideAdminApplication, fetchAdminApplications, fetchAdminPartners, setAdminPartnerStatus,
  type AdminApplication, type AdminPartner,
} from '@/api/admin'
import { useAdminStore } from '@/stores/admin'
import { adminStatusTagType } from '@/utils/adminFormat'
import { formatDateTime } from '@/utils/format'

const admin = useAdminStore()
const loading = ref(false)
const applications = ref<AdminApplication[]>([])
const partners = ref<AdminPartner[]>([])
const applicationRole = ref('')
const applicationStatus = ref('pending')
const partnerRole = ref('')
const partnerStatus = ref('')
const partnerQuery = ref('')

onMounted(load)

async function load() {
  loading.value = true
  await Promise.all([loadApplications(), loadPartners()])
  loading.value = false
}

async function loadApplications() {
  try {
    applications.value = (await fetchAdminApplications(applicationStatus.value, applicationRole.value)).applications ?? []
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '查询申请失败')
  }
}

async function loadPartners() {
  try {
    partners.value = (await fetchAdminPartners(partnerRole.value, partnerStatus.value, partnerQuery.value.trim())).partners ?? []
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '查询合作伙伴失败')
  }
}

async function requireReason(title: string) {
  const { value } = await ElMessageBox.prompt('请输入原因，该操作会写入管理记录。', title, {
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    inputValidator: (input: string) => Boolean(input.trim()) || '原因必填',
  })
  return value.trim()
}

async function decide(row: AdminApplication, approve: boolean) {
  const reason = await requireReason(approve ? '通过申请' : '拒绝申请').catch(() => null)
  if (!reason) return
  try {
    await decideAdminApplication(row.id, approve, reason)
    ElMessage.success(approve ? '申请已通过' : '申请已拒绝')
    await load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '处理失败')
  }
}

async function togglePartner(row: AdminPartner) {
  const freeze = row.status === 'active'
  const reason = await requireReason(freeze ? '冻结合作伙伴' : '解冻合作伙伴').catch(() => null)
  if (!reason) return
  try {
    await setAdminPartnerStatus(row.user_id, freeze, reason)
    ElMessage.success(freeze ? '已冻结' : '已解冻')
    await loadPartners()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '操作失败')
  }
}
</script>

<style scoped>
h1 { margin: 0 0 16px; font-size: 24px; }
.block { margin-bottom: 18px; }
.card-heading { display: flex; justify-content: space-between; align-items: center; }
.muted { color: #6b7280; }
</style>
