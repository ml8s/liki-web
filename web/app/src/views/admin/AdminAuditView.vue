<template>
  <section>
    <div class="page-heading">
      <h1>管理记录</h1>
      <el-button @click="load">
        刷新
      </el-button>
    </div>
    <el-table
      v-loading="loading"
      :data="actions"
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
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchAdminActions, type AdminAction } from '@/api/admin'
import { formatDateTime } from '@/utils/format'

const loading = ref(false)
const actions = ref<AdminAction[]>([])

onMounted(load)

async function load() {
  loading.value = true
  try {
    actions.value = (await fetchAdminActions()).actions ?? []
  } catch (requestError) {
    ElMessage.error(requestError instanceof Error ? requestError.message : '查询管理记录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page-heading { display: flex; justify-content: space-between; align-items: center; }
h1 { margin: 0 0 16px; font-size: 24px; }
</style>
