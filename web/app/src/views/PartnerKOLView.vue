<template>
  <PartnerPageShell
    :title="$t('partner.kol.title')"
    :subtitle="$t('partner.kol.subtitle')"
    :loading="loading"
    :loaded="loaded"
    :error="error"
    :show-refresh="showRefresh"
    @refresh="load(fetchActive)"
  >
    <template v-if="roleStatus === 'active'">
      <PartnerReferralCard
        :title="$t('partner.referralLink')"
        :referral-link="referralLink"
        input-name="kol-referral-link"
        :stats="overviewStats"
        @copy="copy(referralLink)"
      />

      <el-card
        class="section-card"
        shadow="never"
      >
        <template #header>
          <PartnerCardHeader>
            {{ $t('partner.commissions') }}
          </PartnerCardHeader>
        </template>
        <template v-if="commissions.length">
          <el-table
            v-if="!isMobile"
            :data="commissions"
          >
            <el-table-column
              :label="$t('partner.product')"
              min-width="140"
            >
              <template #default="{ row }">
                {{ productName(row.product) }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.orderAmount')"
              width="130"
            >
              <template #default="{ row }">
                {{ formatMinorUnit(row.amount_minor, row.currency) }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.commission')"
              width="130"
            >
              <template #default="{ row }">
                {{ formatMinorUnit(row.referrer_commission_minor, row.currency) }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.status')"
              width="120"
            >
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="commissionTagType(row.status)"
                >
                  {{ commissionLabel(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.time')"
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
              v-for="row in commissions"
              :key="row.id"
              class="record-card"
            >
              <div class="record-card__top">
                <span class="record-card__title">{{ productName(row.product) }}</span>
                <el-tag
                  size="small"
                  :type="commissionTagType(row.status)"
                >
                  {{ commissionLabel(row.status) }}
                </el-tag>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.orderAmount') }}</span><span>{{ formatMinorUnit(row.amount_minor, row.currency) }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.commission') }}</span><span>{{ formatMinorUnit(row.referrer_commission_minor, row.currency) }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.time') }}</span><span>{{ formatDateTime(row.created_at) }}</span>
              </div>
            </div>
          </div>
        </template>
        <el-empty
          v-else
          :description="$t('partner.noCommissions')"
        >
          <el-button
            :disabled="!referralLink"
            @click="copy(referralLink)"
          >
            {{ $t('partner.copyReferralLink') }}
          </el-button>
        </el-empty>
      </el-card>

      <PartnerPayoutCard
        :payouts="payouts"
        :is-mobile="isMobile"
        :submitting="submitting"
        :role-name="roleName"
        @request="submitPayout"
      />
    </template>

    <PartnerRoleStatus
      v-else
      v-model:note="form.note"
      role="kol"
      :role-name="roleName"
      :role-status="roleStatus"
      :other-role-assigned="otherRoleAssigned"
      :application="application"
      :submitting="submitting"
      @submit="submit"
    />
  </PartnerPageShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage } from '@/utils/errors'
import PartnerPageShell from '@/components/partner/PartnerPageShell.vue'
import PartnerReferralCard from '@/components/partner/PartnerReferralCard.vue'
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import PartnerPayoutCard from '@/components/partner/PartnerPayoutCard.vue'
import PartnerRoleStatus from '@/components/partner/PartnerRoleStatus.vue'
import { type PartnerStat } from '@/components/partner/PartnerStatGrid.vue'
import { usePartnerRolePage } from '@/composables/usePartnerRolePage'
import {
  createKOLPayout,
  fetchKOLCommissions,
  fetchKOLPayouts,
  fetchKOLSession,
  type BalanceSummary,
  type KOLOverview,
  type PartnerCommission,
  type PayoutMethod,
} from '@/api/partner'
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'
import { buildReferralLink } from '@/utils/share'

const { t } = useI18n()
const {
  loading, loaded, submitting, error,
  roleStatus, otherRoleAssigned, application,
  payouts, form, isMobile,
  roleName, showRefresh,
  load, submit, copy,
  commissionLabel, commissionTagType,
} = usePartnerRolePage('kol')

const overview = ref<KOLOverview | null>(null)
const commissions = ref<PartnerCommission[]>([])
const balances = ref<BalanceSummary[]>([])

const referralLink = computed(() => {
  const code = overview.value?.partner.code
  return code ? buildReferralLink(code) : ''
})

const overviewStats = computed<PartnerStat[]>(() => [
  { key: 'attributed-users', label: t('partner.attributedUsers'), value: overview.value?.attributed_users ?? 0 },
  ...balances.value.map(balance => ({
    key: `balance-${balance.currency}`,
    label: t('partner.availableBalance', { currency: balance.currency }),
    value: formatMinorUnit(balance.total_minor, balance.currency),
  })),
])

async function fetchActive() {
  const [session, commissionResult, payoutResult] = await Promise.all([
    fetchKOLSession(),
    fetchKOLCommissions(),
    fetchKOLPayouts(),
  ])
  overview.value = session.overview
  commissions.value = commissionResult.commissions
  balances.value = commissionResult.balances
  payouts.value = payoutResult.payouts
}

async function submitPayout(payload: { method: PayoutMethod; account: string }) {
  submitting.value = true
  try {
    const result = await createKOLPayout(payload)
    payouts.value.unshift(result.payout)
    ElMessage.success(t('partner.payoutSubmitted'))
    await load(fetchActive, { skipRoleState: true })
  } catch (err) {
    ElMessage.error(errorMessage(err) || t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => load(fetchActive))
</script>

<style scoped>
@import '@/styles/partner-shared.css';
</style>
