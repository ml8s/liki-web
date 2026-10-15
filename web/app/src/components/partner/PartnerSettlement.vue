<template>
  <el-card shadow="never">
    <template #header>
      <PartnerCardHeader>
        {{ $t('partner.settlement.title', { role: title }) }}
        <template #actions>
          <el-button
            type="primary"
            size="small"
            @click="payoutVisible = true"
          >
            {{ $t('partner.settlement.requestPayout') }}
          </el-button>
        </template>
      </PartnerCardHeader>
    </template>

    <el-alert
      :title="$t('partner.settlement.retiredNotice')"
      type="info"
      :closable="false"
      show-icon
      class="rules"
    />
    <el-alert
      :title="$t('partner.payoutRules')"
      type="info"
      :closable="false"
      show-icon
      class="rules"
    />

    <el-skeleton
      v-if="loading"
      :rows="4"
      animated
      aria-busy="true"
    />
    <el-alert
      v-else-if="error"
      type="error"
      :title="error"
      show-icon
      :closable="false"
    />
    <template v-else>
      <PartnerStatGrid
        class="balance-stats"
        :stats="balanceStats"
      />

      <el-table
        :data="settlement?.payouts ?? []"
        :empty-text="$t('partner.noPayouts')"
        class="payouts"
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
    </template>

    <PayoutDialog
      v-model="payoutVisible"
      :title="$t('partner.settlement.dialogTitle', { role: title })"
      :submitting="submitting"
      @submit="submit"
    />
  </el-card>
</template>

<script setup lang="ts">
import { errorMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import PayoutDialog from '@/components/partner/PayoutDialog.vue'
import PartnerStatGrid, { type PartnerStat } from '@/components/partner/PartnerStatGrid.vue'
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import {
  fetchKOLSettlement,
  fetchScoutSettlement,
  createKOLPayout,
  createScoutPayout,
  type PartnerRole,
  type Payout,
  type PayoutMethod,
  type PayoutStatus,
  type Settlement,
} from '@/api/partner'
import { formatDateTime, formatMinorUnit } from '@/utils/format'
import { partnerPayoutStatus as payoutMeta } from '@/utils/partnerStatus'

const props = defineProps<{ role: PartnerRole }>()
const { t } = useI18n()

const loading = ref(true)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const settlement = ref<Settlement | null>(null)

const title = computed(() => props.role === 'kol' ? 'KOL' : 'Scout')
const balanceStats = computed<PartnerStat[]>(() => (settlement.value?.balances ?? []).map(balance => ({
  key: balance.currency,
  label: t('partner.settlement.balance', { currency: balance.currency }),
  value: formatMinorUnit(balance.total_minor, balance.currency),
})))

async function load() {
  loading.value = true
  error.value = ''
  try {
    settlement.value = props.role === 'kol'
      ? await fetchKOLSettlement()
      : await fetchScoutSettlement()
  } catch (err) {
    error.value = errorMessage(err) || t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

async function submit(payload: { method: PayoutMethod; account: string }) {
  submitting.value = true
  try {
    const result = props.role === 'kol' ? await createKOLPayout({ ...payload }) : await createScoutPayout({ ...payload })
    settlement.value?.payouts.unshift(result.payout)
    payoutVisible.value = false
    ElMessage.success(t('partner.payoutSubmitted'))
    await load()
  } catch (err) {
    ElMessage.error(errorMessage(err) || t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function payoutLabel(status: Payout['status']) {
  return payoutMeta(status).label
}

function payoutTagType(status: PayoutStatus) {
  return payoutMeta(status).tag
}

onMounted(load)
</script>

<style scoped>
.rules { margin-bottom: 14px; }
.balance-stats { margin-bottom: var(--space-4); }
</style>
