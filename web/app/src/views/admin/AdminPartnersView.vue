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
            @click="reloadApplications"
          >
            刷新
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="submitApplicationFilters"
      >
        <el-form-item label="角色">
          <el-select
            v-model="draft.applicationRole"
            class="admin-filter-control"
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
            v-model="draft.applicationStatus"
            class="admin-filter-control"
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
          @click="submitApplicationFilters"
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
          v-if="!isMobile"
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
          v-if="isMobile && !applications.loading.value && !applications.data.value.length"
        >
          <el-empty description="暂无合作申请" />
        </div>
        <div v-if="isMobile">
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

        <el-pagination
          v-if="(applications.meta.value?.total ?? 0) > 0"
          class="admin-pagination"
          :current-page="pagesState.applicationPage"
          :page-size="applications.meta.value?.page_size ?? 100"
          :total="applications.meta.value?.total ?? 0"
          layout="prev, pager, next, total"
          background
          hide-on-single-page
          @current-change="setApplicationPage"
        />
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="card-heading">
          合作伙伴
          <el-button
            size="small"
            :loading="partners.loading.value"
            @click="reloadPartners"
          >
            筛选
          </el-button>
        </div>
      </template>
      <el-form
        inline
        @submit.prevent="submitPartnerFilters"
      >
        <el-form-item label="角色">
          <el-select
            v-model="draft.partnerRole"
            class="admin-filter-control"
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
            v-model="draft.partnerStatus"
            class="admin-filter-control"
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
            v-model="draft.q"
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
          v-if="!isMobile"
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
          v-if="isMobile && !partners.loading.value && !partners.data.value.length"
        >
          <el-empty description="暂无合作伙伴" />
        </div>
        <div v-if="isMobile">
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

        <el-pagination
          v-if="(partners.meta.value?.total ?? 0) > 0"
          class="admin-pagination"
          :current-page="pagesState.partnerPage"
          :page-size="partners.meta.value?.page_size ?? 100"
          :total="partners.meta.value?.total ?? 0"
          layout="prev, pager, next, total"
          background
          hide-on-single-page
          @current-change="setPartnerPage"
        />
      </div>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  decideAdminApplication, fetchAdminApplications, fetchAdminPartners, setAdminPartnerStatus,
  type AdminApplication, type AdminPartner,
} from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { useAdminPagedResource, useAdminViewport, type AdminPagedResource } from '@/composables/useAdminResource'
import { adminRoleLabel } from '@/utils/adminFormat'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { queryChoice, queryText } from '@/utils/query'
import { formatDateTime } from '@/utils/format'

const isMobile = useAdminViewport()

const admin = useAdminStore()

interface PartnerFilters {
  [key: string]: string
  applicationRole: string
  applicationStatus: string
  partnerRole: string
  partnerStatus: string
  q: string
}

type PartnerQueryState = ReturnType<typeof useAdminQueryState<PartnerFilters>>

const partnerDefaults = {
  applicationRole: '',
  applicationStatus: 'pending',
  partnerRole: '',
  partnerStatus: '',
  q: '',
}
const applications: AdminPagedResource<AdminApplication> = useAdminPagedResource<AdminApplication>(async page => {
  const { applicationStatus, applicationRole } = query.applied.value
  return await fetchAdminApplications(applicationStatus, applicationRole, page)
})
const partners: AdminPagedResource<AdminPartner> = useAdminPagedResource<AdminPartner>(async page => {
  const { partnerStatus, partnerRole, q } = query.applied.value
  return await fetchAdminPartners(partnerRole, partnerStatus, q.trim(), page)
})

const query: PartnerQueryState = useAdminQueryState<PartnerFilters>({
  defaults: partnerDefaults,
  pageKeys: ['applicationPage', 'partnerPage'],
  decode: query => ({
    applicationRole: queryChoice(query, 'applicationRole', ['', 'kol', 'scout'], partnerDefaults.applicationRole),
    applicationStatus: queryChoice(query, 'applicationStatus', ['', 'pending', 'approved', 'rejected'], partnerDefaults.applicationStatus),
    partnerRole: queryChoice(query, 'partnerRole', ['', 'kol', 'scout'], partnerDefaults.partnerRole),
    partnerStatus: queryChoice(query, 'partnerStatus', ['', 'active', 'frozen'], partnerDefaults.partnerStatus),
    q: queryText(query, 'q'),
  }),
  encode: (filters, pages) => ({
    applicationRole: filters.applicationRole || undefined,
    applicationStatus: filters.applicationStatus === partnerDefaults.applicationStatus ? undefined : filters.applicationStatus,
    partnerRole: filters.partnerRole || undefined,
    partnerStatus: filters.partnerStatus || undefined,
    q: filters.q || undefined,
    applicationPage: pages.applicationPage === 1 ? undefined : pages.applicationPage,
    partnerPage: pages.partnerPage === 1 ? undefined : pages.partnerPage,
  }),
  onApplied: async (_filters, pages): Promise<void> => {
    await Promise.all([
      applications.load(pages.applicationPage),
      partners.load(pages.partnerPage),
    ])
  },
})

const { draft } = query
const pagesState = query.pages
void query.initialize()

function submitApplicationFilters() {
  void query.submit({ applicationPage: 1 })
}

function submitPartnerFilters() {
  void query.submit({ partnerPage: 1 })
}

function reloadApplications() {
  void applications.load(pagesState.value.applicationPage)
}

function reloadPartners() {
  void partners.load(pagesState.value.partnerPage)
}

function setApplicationPage(page: number) {
  void query.setPage('applicationPage', page)
}

function setPartnerPage(page: number) {
  void query.setPage('partnerPage', page)
}

async function requireReason(title: string) {
  const { value } = await ElMessageBox.prompt('请输入原因，该操作会写入管理记录。', title, {
    confirmButtonText: title,
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
    await applications.load(pagesState.value.applicationPage)
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
    await partners.load(pagesState.value.partnerPage)
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '操作失败')
  }
}
</script>

<style scoped>
.block { margin-bottom: var(--space-5); }
</style>
