<template>
  <section>
    <AdminPageHeader
      title="管理总览"
      :description="`当前账号：${admin.email}。菜单和接口都按能力独立授权。`"
    />
    <div class="capability-grid">
      <el-card
        v-for="capability in admin.capabilities"
        :key="capability"
        shadow="never"
      >
        <el-tag size="small">
          {{ capabilityLabel(capability) }}
        </el-tag>
        <p class="mono">
          {{ capability }}
        </p>
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import { useAdminStore } from '@/stores/admin'
import { capabilityLabel } from '@/utils/adminFormat'

const admin = useAdminStore()
</script>

<style scoped>
.capability-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: var(--space-3); margin-top: var(--space-4); }
.capability-grid p {
  margin: var(--space-2) 0 0;
  color: var(--liki-text-2);
  font-size: var(--font-size-caption);
}
</style>
