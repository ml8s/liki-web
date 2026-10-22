<template>
  <el-card shadow="never">
    <template #header>
      <PartnerCardHeader>
        {{ $t('partner.settlement.title', { role: title }) }}
        <template #actions>
          <el-button
            v-if="error && settlement"
            size="small"
            @click="load"
          >
            {{ $t('common.retry') }}
          </el-button>
          <el-button
            type="primary"
            size="small"
            :disabled="!ready"
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
      v-if="loading && !settlement"
      :rows="4"
      animated
      aria-busy="true"
    />
    <el-result
      v-else-if="error && !settlement"
      icon="error"
      :title="$t('common.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="load"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>
    <template v-else>
      <el-alert
        v-if="error"
        type="error"
        :title="error"
        show-icon
        :closable="false"
        class="rules"
      />

      <div
        v-loading="loading"
        :aria-busy="loading"
      >
        <PartnerStatGrid
          class="balance-stats"
          :stats="balanceStats"
        />

        <template v-if="settlement && settlement.payouts.length">
          <el-table
            v-if="!isMobile"
            :data="settlement.payouts"
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
              v-for="row in settlement.payouts"
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
            :disabled="!ready"
            @click="payoutVisible = true"
          >
            {{ $t('partner.settlement.requestPayout') }}
          </el-button>
        </el-empty>
      </div>
    </template>

    <PartnerPayoutDialog
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
import { useI18n } from 'vue-i18n'
import PartnerPayoutDialog from '@/components/partner/PartnerPayoutDialog.vue'
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
import { useMobile } from '@/composables/useMobile'
import { partnerPayoutStatus as payoutMeta, partnerRoleName } from '@/utils/partnerStatus'

const props = defineProps<{ role: PartnerRole }>()
const { t } = useI18n()

const loading = ref(true)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const settlement = ref<Settlement | null>(null)
const isMobile = useMobile(480)

const title = computed(() => partnerRoleName(props.role))
const ready = computed(() => !loading.value && !error.value && Boolean(settlement.value))
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
.rules { margin-bottom: var(--space-4); }
.balance-stats { margin-bottom: var(--space-4); }
.record-cards { display: grid; gap: var(--space-3); }
.record-card {
  padding: var(--space-4);
  border: 1px solid var(--liki-line);
  border-radius: var(--radius-lg);
  background: var(--liki-surface);
}
.record-card__top { display: flex; align-items: center; justify-content: space-between; gap: var(--space-2); margin-bottom: var(--space-2); }
.record-card__title { font-weight: 600; font-size: var(--font-size-label); word-break: break-all; }
.record-row { display: flex; gap: var(--space-2); padding: var(--space-1) 0; font-size: var(--font-size-body); }
.record-row .k { flex-shrink: 0; min-width: 4em; color: var(--liki-text-2); }
.record-row span:last-child { word-break: break-all; }
</style>
