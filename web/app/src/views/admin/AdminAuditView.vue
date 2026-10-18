<template>
  <section>
    <AdminPageHeader title="合作伙伴审计">
      <template #actions>
        <el-button
          :loading="actions.loading.value"
          @click="reloadCurrentPage"
        >
          刷新
        </el-button>
      </template>
    </AdminPageHeader>
    <el-alert
      v-if="actions.error.value"
      :title="actions.error.value"
      type="error"
      show-icon
      :closable="false"
      class="query-error"
    />
    <div
      v-loading="actions.loading.value"
      :aria-busy="actions.loading.value"
    >
      <el-table
        v-if="!isMobile"
        :data="actions.data.value"
        row-key="id"
      >
        <el-table-column
          label="时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatAdminDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column
          prop="actor"
          label="操作人"
          min-width="180"
          show-overflow-tooltip
        />
        <el-table-column
          prop="action"
          label="动作"
          min-width="170"
        />
        <el-table-column
          prop="target_id"
          label="对象"
          min-width="210"
          show-overflow-tooltip
        />
        <el-table-column
          prop="reason"
          label="原因"
          min-width="240"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            {{ row.reason || '-' }}
          </template>
        </el-table-column>
      </el-table>

      <div
        v-if="isMobile && !actions.loading.value && !actions.data.value.length"
      >
        <el-empty description="暂无合作伙伴审计记录" />
      </div>
      <div v-if="isMobile">
        <AdminRecordCard
          v-for="row in actions.data.value"
          :key="row.id"
          :fields="[
            { key: 'created_at', label: '时间', value: formatAdminDateTime(row.created_at) },
            { key: 'actor', label: '操作人', value: row.actor },
            { key: 'action', label: '动作', value: row.action },
            { key: 'target_id', label: '对象', value: row.target_id, kind: 'mono' },
            { key: 'reason', label: '原因', value: row.reason || '-' },
          ]"
        />
      </div>

      <el-pagination
        v-if="(actions.meta.value?.total ?? 0) > 0"
        class="admin-pagination"
        :current-page="pages.page"
        :page-size="actions.meta.value?.page_size ?? 100"
        :total="actions.meta.value?.total ?? 0"
        layout="prev, pager, next, total"
        background
        hide-on-single-page
        @current-change="setActionPage"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { fetchAdminActions, type AdminAction } from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import { useAdminPagedResource, useAdminViewport } from '@/composables/useAdminResource'
import { useAdminQueryState } from '@/composables/useAdminQueryState'
import { formatAdminDateTime } from '@/utils/adminFormat'

const isMobile = useAdminViewport()

interface AuditFilters {
  [key: string]: string
}

const actions = useAdminPagedResource<AdminAction>(async page => await fetchAdminActions(page))

const query = useAdminQueryState<AuditFilters>({
  defaults: {},
  pageKeys: ['page'],
  decode: () => ({}),
  encode: (_filters, pages) => ({ page: pages.page === 1 ? undefined : pages.page }),
  onApplied: (_filters, pages): Promise<void> => actions.load(pages.page),
})

const { pages } = query
void query.initialize()

function reloadCurrentPage() {
  void actions.load(pages.value.page)
}

function setActionPage(page: number) {
  void query.setPage('page', page)
}
</script>
