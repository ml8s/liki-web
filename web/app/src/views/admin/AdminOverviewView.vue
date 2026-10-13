<template>
  <section>
    <h1>管理总览</h1>
    <p class="muted">
      当前账号：{{ admin.email }}。菜单和接口都按能力独立授权。
    </p>
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
import { useAdminStore } from '@/stores/admin'
import { capabilityLabel } from '@/utils/adminFormat'

const admin = useAdminStore()
</script>

<style scoped>
h1 { margin: 0 0 8px; font-size: 24px; }
.muted { color: var(--liki-text-muted, #6b7280); margin-top: 0; }
.capability-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(190px, 1fr)); gap: 12px; margin-top: 18px; }
.capability-grid p { margin: 10px 0 0; font-size: 12px; color: #6b7280; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>
