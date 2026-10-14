<template>
  <section>
    <div class="page-heading">
      <h1>合作伙伴审计</h1>
      <el-button
        :loading="actions.loading.value"
        @click="actions.load"
      >
        刷新
      </el-button>
    </div>
    <el-alert
      v-if="actions.error.value"
      :title="actions.error.value"
      type="error"
      show-icon
      :closable="false"
      class="query-error"
    />
    <el-table
      v-loading="actions.loading.value"
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
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { fetchAdminActions, type AdminAction } from '@/api/admin'
import { useAdminResource } from '@/composables/useAdminResource'
import { formatDateTime } from '@/utils/format'

const actions = useAdminResource<AdminAction>(async () => {
  const result = await fetchAdminActions()
  return result.actions ?? []
})

onMounted(actions.load)
</script>

<style scoped>
.page-heading { display: flex; justify-content: space-between; align-items: center; }
h1 { margin: 0 0 16px; font-size: 24px; }
.query-error { margin-bottom: 12px; }
</style>
