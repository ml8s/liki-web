<template>
  <section>
    <AdminPageHeader title="合作伙伴审计">
      <template #actions>
        <el-button
          :loading="actions.loading.value"
          @click="actions.load(1)"
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
        class="admin-desktop-only"
        :data="actions.data.value"
        row-key="id"
      >
        <el-table-column
          label="时间"
          width="180"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
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
        v-if="!actions.loading.value && !actions.data.value.length"
        class="admin-mobile-only"
      >
        <el-empty description="暂无合作伙伴审计记录" />
      </div>
      <div class="admin-mobile-only">
        <AdminRecordCard
          v-for="row in actions.data.value"
          :key="row.id"
          :fields="[
            { key: 'created_at', label: '时间', value: formatDateTime(row.created_at) },
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
        :current-page="actionPage"
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
import { onMounted, ref } from 'vue'
import { fetchAdminActions, type AdminAction } from '@/api/admin'
import AdminRecordCard from '@/components/admin/AdminRecordCard.vue'
import { useAdminPagedResource } from '@/composables/useAdminResource'
import { formatDateTime } from '@/utils/format'

const actionPage = ref(1)
const actions = useAdminPagedResource<AdminAction>(async page => await fetchAdminActions(page))

onMounted(() => actions.load(actionPage.value))

function setActionPage(page: number) {
  actionPage.value = page
  void actions.load(page)
}
</script>
