<template>
  <section>
    <AdminPageHeader title="合作伙伴" />
    <el-card
      shadow="never"
      class="block"
    >
      <template #header>
        <div class="card-heading">
          合作申请
          <el-button
            size="small"
            :loading="applications.loading.value"
            @click="applications.load"
          >
            刷新
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="applications.load"
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
          @click="applications.load"
        >
          查询
        </el-button>
      </el-form>
      <el-alert
        v-if="applications.error.value"
        :title="applications.error.value"
        type="error"
        show-icon
        :closable="false"
        class="query-error"
      />
      <div
        v-loading="applications.loading.value"
        :aria-busy="applications.loading.value"
      >
        <el-table
          class="admin-desktop-only"
          :data="applications.data.value"
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
              <AdminStatusTag :status="row.status" />
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

        <div
          v-if="!applications.loading.value && !applications.data.value.length"
          class="admin-mobile-only"
        >
          <el-empty description="暂无合作申请" />
        </div>
        <div class="admin-mobile-only">
          <AdminRecordCard
            v-for="row in applications.data.value"
            :key="row.id"
            :fields="[
              { key: 'code', label: '推广码', value: row.code },
              { key: 'role', label: '角色', value: adminRoleLabel(row.type) },
              { key: 'note', label: '说明', value: row.note },
              { key: 'status', label: '状态', value: row.status, kind: 'status' },
              { key: 'created', label: '时间', value: formatDateTime(row.created_at) },
            ]"
          >
            <template
              v-if="admin.can('partner.approve') && row.status === 'pending'"
              #actions
            >
              <el-button
                size="small"
                type="primary"
                @click="decide(row, true)"
              >
                通过
              </el-button>
              <el-button
                size="small"
                type="danger"
                plain
                @click="decide(row, false)"
              >
                拒绝
              </el-button>
            </template>
          </AdminRecordCard>
        </div>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="card-heading">
          合作伙伴
          <el-button
            size="small"
            :loading="partners.loading.value"
            @click="partners.load"
          >
            筛选
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="partners.load"
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
            name="partner-search"
            autocomplete="off"
            spellcheck="false"
            placeholder="ID / 推广码"
            clearable
          />
        </el-form-item>
      </el-form>
      <div
        v-loading="partners.loading.value"
        :aria-busy="partners.loading.value"
      >
        <el-table
          class="admin-desktop-only"
          :data="partners.data.value"
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
              <AdminStatusTag :status="row.status" />
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

        <div
          v-if="!partners.loading.value && !partners.data.value.length"
          class="admin-mobile-only"
        >
          <el-empty description="暂无合作伙伴" />
        </div>
        <div class="admin-mobile-only">
          <AdminRecordCard
            v-for="row in partners.data.value"
            :key="row.user_id"
            :fields="[
              { key: 'code', label: '推广码', value: row.code },
              { key: 'user_id', label: 'User ID', value: row.user_id, kind: 'mono' },
              { key: 'role', label: '角色', value: adminRoleLabel(row.role) },
              { key: 'status', label: '状态', value: row.status, kind: 'status' },
              { key: 'created', label: '时间', value: formatDateTime(row.created_at) },
            ]"
          >
            <template
              v-if="admin.canAny(['partner.freeze','partner.unfreeze'])"
              #actions
            >
              <el-button
                size="small"
                :type="row.status === 'active' ? 'danger' : 'primary'"
                plain
                :disabled="row.status === 'active' ? !admin.can('partner.freeze') : !admin.can('partner.unfreeze')"
                @click="togglePartner(row)"
              >
                {{ row.status === 'active' ? '冻结' : '解冻' }}
              </el-button>
            </template>
          </AdminRecordCard>
        </div>
      </div>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  decideAdminApplication, fetchAdminApplications, fetchAdminPartners, setAdminPartnerStatus,
  type AdminApplication, type AdminPartner,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { useRoute, useRouter } from 'vue-router'
import { useAdminResource } from '@/composables/useAdminResource'
import { adminRoleLabel } from '@/utils/adminFormat'
import { queryChoice, queryText } from '@/utils/query'
import { formatDateTime } from '@/utils/format'

const admin = useAdminStore()
const route = useRoute()
const router = useRouter()
const applicationRole = ref(queryChoice(route.query, 'applicationRole', ['', 'kol', 'scout'], ''))
const applicationStatus = ref(queryChoice(route.query, 'applicationStatus', ['', 'pending', 'approved', 'rejected'], 'pending'))
const partnerRole = ref(queryChoice(route.query, 'partnerRole', ['', 'kol', 'scout'], ''))
const partnerStatus = ref(queryChoice(route.query, 'partnerStatus', ['', 'active', 'frozen'], ''))
const partnerQuery = ref(queryText(route.query, 'q'))
const applications = useAdminResource<AdminApplication>(async () => {
  const result = await fetchAdminApplications(applicationStatus.value, applicationRole.value)
  return result.applications ?? []
})
const partners = useAdminResource<AdminPartner>(async () => {
  const result = await fetchAdminPartners(partnerRole.value, partnerStatus.value, partnerQuery.value.trim())
  return result.partners ?? []
})
onMounted(() => {
  void Promise.all([applications.load(), partners.load()])
})

watch(
  [applicationRole, applicationStatus, partnerRole, partnerStatus, partnerQuery],
  async ([applicationRole, applicationStatus, partnerRole, partnerStatus, partnerQuery]) => {
    await router.replace({
      query: {
        ...route.query,
        applicationRole,
        applicationStatus,
        partnerRole,
        partnerStatus,
        q: partnerQuery || undefined,
      },
    })
  },
)

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
    await applications.load()
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
    await partners.load()
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '操作失败')
  }
}
</script>

<style scoped>
.block { margin-bottom: var(--space-5); }
</style>
