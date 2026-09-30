<template>
  <el-card shadow="never">
    <template #header>
      <div class="head">
        <span>{{ $t('partner.settlement.title', { role: title }) }}</span>
        <el-button
          type="primary"
          size="small"
          @click="payoutVisible = true"
        >
          {{ $t('partner.settlement.requestPayout') }}
        </el-button>
      </div>
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
      <div class="stat-grid">
        <div
          v-for="balance in settlement?.balances ?? []"
          :key="balance.currency"
          class="stat"
        >
          <span class="label">{{ $t('partner.settlement.balance', { currency: balance.currency }) }}</span>
          <strong>{{ formatMinorUnit(balance.total_minor, balance.currency) }}</strong>
        </div>
      </div>

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
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import PayoutDialog from '@/components/partner/PayoutDialog.vue'
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
} from '@/api/referral'
import { formatDateTime, formatMinorUnit } from '@/utils/format'

const props = defineProps<{ role: PartnerRole }>()
const { t } = useI18n()

const loading = ref(true)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const settlement = ref<Settlement | null>(null)

const title = computed(() => props.role === 'kol' ? 'KOL' : 'Scout')

async function load() {
  loading.value = true
  error.value = ''
  try {
    settlement.value = props.role === 'kol'
      ? await fetchKOLSettlement()
      : await fetchScoutSettlement()
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.loadFailed')
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
    ElMessage.error(err instanceof Error ? err.message : t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function payoutLabel(status: Payout['status']) {
  const labels: Record<Payout['status'], string> = {
    requested: t('partner.payoutStatus.requested'),
    pending: t('partner.payoutStatus.pending'),
    approved: t('partner.payoutStatus.approved'),
    rejected: t('partner.payoutStatus.rejected'),
    paid: t('partner.payoutStatus.paid'),
    failed: t('partner.payoutStatus.failed'),
  }
  return labels[status]
}

function payoutTagType(status: PayoutStatus) {
  if (status === 'paid') return 'success'
  if (status === 'rejected' || status === 'failed') return 'danger'
  return 'warning'
}

onMounted(load)
</script>

<style scoped>
.head { display: flex; align-items: center; justify-content: space-between; font-weight: 600; }
.rules { margin-bottom: 14px; }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; margin-bottom: 18px; }
.stat { border: 1px solid var(--liki-line); border-radius: 12px; padding: 14px; background: var(--liki-paper); }
.stat .label { display: block; margin-bottom: 6px; color: var(--liki-text-2); font-size: 13px; }
.stat strong { font-size: 20px; }
</style>
