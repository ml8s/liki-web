<template>
  <el-card
    class="admin-record-card"
    shadow="never"
  >
    <el-descriptions
      :column="1"
      border
      size="small"
    >
      <el-descriptions-item
        v-for="field in fields"
        :key="field.key"
        :label="field.label"
      >
        <slot
          name="field"
          :field="field"
        >
          <AdminStatusTag
            v-if="field.kind === 'status' && statusDomain"
            :domain="statusDomain"
            :status="field.value"
          />
          <span
            v-else-if="field.kind === 'money'"
            class="admin-record-strong"
          >
            {{ formatAdminMoney(Number(field.value || 0), field.currency || '') }}
          </span>
          <span
            v-else
            :class="{ mono: field.kind === 'mono' }"
          >
            {{ field.value || '-' }}
          </span>
        </slot>
      </el-descriptions-item>
    </el-descriptions>
    <el-space
      v-if="$slots.actions"
      wrap
      class="admin-record-actions"
    >
      <slot name="actions" />
    </el-space>
  </el-card>
</template>

<script setup lang="ts">
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import { formatAdminMoney, type AdminStatusDomain } from '@/utils/adminFormat'

export interface AdminRecordField {
  key: string
  label: string
  value: string | number
  kind?: 'text' | 'mono' | 'status' | 'money'
  currency?: string
}

defineProps<{
  fields: AdminRecordField[]
  /** Set whenever a field uses kind: 'status'; a caller that replaces the
   *  field slot renders its own AdminStatusTag instead. */
  statusDomain?: AdminStatusDomain
}>()
</script>
