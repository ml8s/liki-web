<template>
  <PartnerSettlement
    v-if="roleStatus === 'retired'"
    :role="role"
  />
  <el-result
    v-else-if="roleStatus === 'frozen'"
    icon="warning"
    :title="$t('partner.frozenTitle', { role: roleName })"
    :description="$t('partner.frozenDescription')"
  />
  <el-result
    v-else-if="otherRoleAssigned"
    icon="info"
    :title="$t('partner.roleConflictTitle')"
    :description="$t('partner.roleConflictDescription')"
  />
  <el-result
    v-else-if="application?.status === 'pending'"
    icon="info"
    :title="$t('partner.pendingTitle', { role: roleName })"
    :description="$t(`partner.${role}.pendingDescription`)"
  />
  <el-card
    v-else
    class="apply-card"
    shadow="never"
  >
    <el-alert
      v-if="application?.status === 'rejected'"
      :title="rejectionTitle(application.decision_reason)"
      type="warning"
      :closable="false"
      show-icon
      class="rules"
    />
    <h2>{{ $t(`partner.${role}.applyTitle`) }}</h2>
    <p>{{ $t(`partner.${role}.applyDescription`) }}</p>
    <el-form
      label-position="top"
      @submit.prevent="$emit('submit')"
    >
      <el-form-item :label="$t(`partner.${role}.noteLabel`)">
        <el-input
          v-model="note"
          :name="`${role}-application-note`"
          autocomplete="off"
          type="textarea"
          :rows="5"
          maxlength="500"
          show-word-limit
          :placeholder="$t(`partner.${role}.notePlaceholder`)"
        />
      </el-form-item>
      <el-button
        type="primary"
        size="large"
        native-type="submit"
        :loading="submitting"
      >
        {{ $t('partner.submitApplication') }}
      </el-button>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import PartnerSettlement from '@/components/partner/PartnerSettlement.vue'
import { useI18n } from 'vue-i18n'
import type { PartnerApplication, PartnerRoleStatus } from '@/api/partner'

defineProps<{
  role: 'kol' | 'scout'
  roleName: string
  roleStatus: PartnerRoleStatus | ''
  otherRoleAssigned: boolean
  application: PartnerApplication | null
  submitting: boolean
}>()

const note = defineModel<string>('note', { default: '' })

defineEmits<{ submit: [] }>()

const { t } = useI18n()

function rejectionTitle(reason?: string) {
  return reason ? t('partner.rejectedTitle', { reason }) : t('partner.rejectedTitleNoReason')
}
</script>
