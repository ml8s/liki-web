<template>
  <article class="admin-record-card">
    <dl>
      <div
        v-for="field in fields"
        :key="field.key"
        class="admin-record-row"
      >
        <dt>{{ field.label }}</dt>
        <dd>
          <slot
            name="field"
            :field="field"
          >
            <AdminStatusTag
              v-if="field.kind === 'status'"
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
        </dd>
      </div>
    </dl>
    <div
      v-if="$slots.actions"
      class="admin-record-actions"
    >
      <slot name="actions" />
    </div>
  </article>
</template>

<script setup lang="ts">
import AdminStatusTag from '@/components/admin/AdminStatusTag.vue'
import { formatAdminMoney } from '@/utils/adminFormat'

export interface AdminRecordField {
  key: string
  label: string
  value: string | number
  kind?: 'text' | 'mono' | 'status' | 'money'
  currency?: string
}

defineProps<{
  fields: AdminRecordField[]
}>()
</script>
