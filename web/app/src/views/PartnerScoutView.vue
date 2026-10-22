<template>
  <PartnerPageShell
    :title="$t('partner.scout.title')"
    :subtitle="$t('partner.scout.subtitle')"
    :loading="loading"
    :loaded="loaded"
    :error="error"
    :show-refresh="showRefresh"
    @refresh="load(fetchActive)"
  >
    <template v-if="roleStatus === 'active'">
      <PartnerReferralCard
        :title="$t('partner.scout.inviteLink')"
        :referral-link="referralLink"
        input-name="scout-referral-link"
        :stats="overviewStats"
        @copy="copy(referralLink)"
      >
        <p class="share-hint">
          {{ $t('partner.scout.shareHint') }}
        </p>
      </PartnerReferralCard>

      <el-card
        class="section-card"
        shadow="never"
      >
        <template #header>
          <PartnerCardHeader>
            {{ $t('partner.scout.activeKols') }}
          </PartnerCardHeader>
        </template>
        <template v-if="recruits.length">
          <el-table
            v-if="!isMobile"
            :data="recruits"
          >
            <el-table-column
              :label="$t('partner.scout.kolCode')"
              min-width="150"
            >
              <template #default="{ row }">
                {{ row.code }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.scout.orderCount')"
              width="120"
              prop="order_count"
            />
            <el-table-column
              :label="$t('partner.scout.joinedAt')"
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
              v-for="row in recruits"
              :key="row.user_id"
              class="record-card"
            >
              <div class="record-card__top">
                <span class="record-card__title">{{ row.code }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.scout.orderCount') }}</span><span>{{ row.order_count }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.scout.joinedAt') }}</span><span>{{ formatDateTime(row.created_at) }}</span>
              </div>
            </div>
          </div>
        </template>
        <el-empty
          v-else
          :description="$t('partner.scout.noRecruits')"
        >
          <el-button
            :disabled="!referralLink"
            @click="copy(referralLink)"
          >
            {{ $t('partner.scout.copyInviteLink') }}
          </el-button>
        </el-empty>
      </el-card>

      <el-card
        class="section-card"
        shadow="never"
      >
        <template #header>
          <PartnerCardHeader>
            {{ $t('partner.scout.channelCommissions') }}
          </PartnerCardHeader>
        </template>
        <template v-if="sales.length">
          <el-table
            v-if="!isMobile"
            :data="sales"
          >
            <el-table-column
              prop="order_ref"
              :label="$t('partner.order')"
              min-width="170"
            />
            <el-table-column
              :label="$t('partner.product')"
              min-width="140"
            >
              <template #default="{ row }">
                {{ productName(row.product) }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.salesAmount')"
              width="130"
            >
              <template #default="{ row }">
                {{ formatMinorUnit(row.amount_minor, row.currency) }}
              </template>
            </el-table-column>
            <el-table-column
              :label="$t('partner.scout.commission')"
              width="130"
            >
              <template #default="{ row }">
                {{ formatMinorUnit(row.scout_commission_minor, row.currency) }}
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
              v-for="row in sales"
              :key="row.order_ref"
              class="record-card"
            >
              <div class="record-card__top">
                <span class="record-card__title">{{ row.order_ref }}</span>
                <el-tag
                  size="small"
                  :type="commissionTagType(row.status)"
                >
                  {{ commissionLabel(row.status) }}
                </el-tag>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.product') }}</span><span>{{ productName(row.product) }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.salesAmount') }}</span><span>{{ formatMinorUnit(row.amount_minor, row.currency) }}</span>
              </div>
              <div class="record-row">
                <span class="k">{{ $t('partner.scout.commission') }}</span><span>{{ formatMinorUnit(row.scout_commission_minor, row.currency) }}</span>
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
            {{ $t('partner.scout.copyInviteLink') }}
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
      role="scout"
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
  createScoutPayout,
  fetchScoutCommissions,
  fetchScoutPayouts,
  fetchScoutRecruits,
  fetchScoutSession,
  type ScoutOverview,
  type ScoutRecruit,
  type ScoutSale,
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
} = usePartnerRolePage('scout')

const overview = ref<ScoutOverview | null>(null)
const recruits = ref<ScoutRecruit[]>([])
const sales = ref<ScoutSale[]>([])

const referralLink = computed(() => {
  const code = overview.value?.partner.code
  return code ? buildReferralLink(code) : ''
})

const overviewStats = computed<PartnerStat[]>(() => [
  { key: 'leads', label: t('partner.scout.leads'), value: overview.value?.lead_count ?? 0 },
  { key: 'active-kols', label: t('partner.scout.activeKols'), value: overview.value?.recruit_count ?? 0 },
  { key: 'channel-orders', label: t('partner.scout.channelOrders'), value: overview.value?.order_count ?? 0 },
  ...(overview.value?.balances ?? []).map(balance => ({
    key: `balance-${balance.currency}`,
    label: t('partner.commissionBalance', { currency: balance.currency }),
    value: formatMinorUnit(balance.commission_minor, balance.currency),
  })),
])

async function fetchActive() {
  const [session, recruitResult, commissionResult, payoutResult] = await Promise.all([
    fetchScoutSession(), fetchScoutRecruits(), fetchScoutCommissions(), fetchScoutPayouts(),
  ])
  overview.value = session.overview
  recruits.value = recruitResult.recruits
  sales.value = commissionResult.sales
  payouts.value = payoutResult.payouts
}

async function submitPayout(payload: { method: PayoutMethod; account: string }) {
  submitting.value = true
  try {
    const result = await createScoutPayout(payload)
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
.share-hint { margin: 10px 0 0; color: var(--liki-text-2); font-size: var(--font-size-body); }
</style>
