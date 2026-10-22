<template>
  <el-card
    class="section-card"
    shadow="never"
  >
    <template #header>
      <PartnerCardHeader>
        {{ $t('partner.payouts') }}
        <template #actions>
          <el-button
            type="primary"
            size="small"
            @click="dialogVisible = true"
          >
            {{ $t('partner.settlement.requestPayout') }}
          </el-button>
        </template>
      </PartnerCardHeader>
    </template>
    <el-alert
      :title="$t('partner.payoutRules')"
      type="info"
      :closable="false"
      show-icon
      class="rules"
    />
    <template v-if="payouts.length">
      <el-table
        v-if="!isMobile"
        :data="payouts"
      >
        <el-table-column
          :label="$t('partner.amount')"
          width="130"
        >
          <template #default="{ row }">
            {{ formatMinorUnit(row.amount_minor, row.currency) }}
          </template>
        </el-table-column>
        <el-table-column
          prop="method"
          :label="$t('partner.method')"
          width="130"
        />
        <el-table-column
          :label="$t('partner.account')"
          min-width="180"
        >
          <template #default="{ row }">
            {{ row.account }}
          </template>
        </el-table-column>
        <el-table-column
          :label="$t('partner.status')"
          width="120"
        >
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="payoutTagType(row.status)"
            >
              {{ payoutLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          :label="$t('partner.requestedAt')"
          width="180"
        >
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
      </el-table>
      <div
        v-else
        class="record-cards"
      >
        <div
          v-for="row in payouts"
          :key="row.id"
          class="record-card"
        >
          <div class="record-card__top">
            <span class="record-card__title">{{ formatMinorUnit(row.amount_minor, row.currency) }}</span>
            <el-tag
              size="small"
              :type="payoutTagType(row.status)"
            >
              {{ payoutLabel(row.status) }}
            </el-tag>
          </div>
          <div class="record-row">
            <span class="k">{{ $t('partner.method') }}</span><span>{{ row.method }}</span>
          </div>
          <div class="record-row">
            <span class="k">{{ $t('partner.account') }}</span><span>{{ row.account }}</span>
          </div>
          <div class="record-row">
            <span class="k">{{ $t('partner.requestedAt') }}</span><span>{{ formatDateTime(row.created_at) }}</span>
          </div>
        </div>
      </div>
    </template>
    <el-empty
      v-else
      :description="$t('partner.noPayouts')"
    >
      <el-button
        type="primary"
        @click="dialogVisible = true"
      >
        {{ $t('partner.settlement.requestPayout') }}
      </el-button>
    </el-empty>
  </el-card>

  <PartnerPayoutDialog
    v-model="dialogVisible"
    :title="$t('partner.payoutTitle', { role: roleName })"
    :submitting="submitting"
    @submit="$emit('request', $event)"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import PartnerPayoutDialog from '@/components/partner/PartnerPayoutDialog.vue'
import { partnerPayoutStatus } from '@/utils/partnerStatus'
import { formatDateTime, formatMinorUnit } from '@/utils/format'
import type { Payout, PayoutMethod } from '@/api/partner'

defineProps<{
  payouts: Payout[]
  isMobile: boolean
  submitting: boolean
  roleName: string
}>()

defineEmits<{ request: [payload: { method: PayoutMethod; account: string }] }>()

const dialogVisible = ref(false)

function payoutLabel(status: Payout['status']) {
  return partnerPayoutStatus(status).label
}

function payoutTagType(status: Payout['status']) {
  return partnerPayoutStatus(status).tag
}
</script>
