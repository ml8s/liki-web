<template>
  <section>
    <AppPageHeader
      :title="$t('partner.kol.title')"
      :subtitle="$t('partner.kol.subtitle')"
    >
      <template #actions>
        <el-button
          :loading="loading"
          @click="load"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-skeleton
      v-if="loading"
      :rows="6"
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

    <template v-else-if="roleStatus === 'active'">
      <el-card
        class="section-card"
        shadow="never"
      >
        <template #header>
          <PartnerCardHeader>
            {{ $t('partner.referralLink') }}
            <template #actions>
              <el-button
                size="small"
                @click="copy(referralLink)"
              >
                {{ $t('common.copy') }}
              </el-button>
            </template>
          </PartnerCardHeader>
        </template>
        <el-input
          v-model="referralLink"
          :aria-label="$t('partner.referralLink')"
          name="kol-referral-link"
          autocomplete="off"
          spellcheck="false"
          readonly
        />
        <PartnerStatGrid
          class="overview-stats"
          :stats="overviewStats"
        />
      </el-card>

      <el-card
        class="section-card"
        shadow="never"
      >
        <template #header>
          <span>{{ $t('partner.commissions') }}</span>
        </template>
        <el-table
          :data="commissions"
          :empty-text="$t('partner.noCommissions')"
        >
          <el-table-column
            prop="product"
            :label="$t('partner.product')"
            min-width="130"
          />
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
            width="110"
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
      </el-card>

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
                @click="payoutVisible = true"
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
        <el-table
          :data="payouts"
          :empty-text="$t('partner.noPayouts')"
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
            width="120"
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
      </el-card>
    </template>

    <PartnerSettlement
      v-else-if="roleStatus === 'retired'"
      role="kol"
    />

    <el-result
      v-else-if="roleStatus === 'frozen'"
      icon="warning"
      :title="$t('partner.frozenTitle', { role: $t('partner.kol.roleName') })"
      :description="$t('partner.frozenDescription')"
    />

    <el-result
      v-else-if="otherRoleAssigned"
      icon="info"
      :title="$t('partner.otherRoleTitle', { role: $t('partner.scout.roleName') })"
      :description="$t('partner.roleConflictDescription')"
    />

    <el-result
      v-else-if="application?.status === 'pending'"
      icon="info"
      :title="$t('partner.pendingTitle', { role: $t('partner.kol.roleName') })"
      :description="$t('partner.kol.pendingDescription')"
    />

    <el-card
      v-else
      class="apply-card"
      shadow="never"
    >
      <el-alert
        v-if="application?.status === 'rejected'"
        :title="$t('partner.rejectedTitle', { reason: application.decision_reason || $t('partner.defaultRejectionReason') })"
        type="warning"
        :closable="false"
        show-icon
        class="rules"
      />
      <h2>{{ $t('partner.kol.applyTitle') }}</h2>
      <p>{{ $t('partner.kol.applyDescription') }}</p>
      <el-form
        label-position="top"
        @submit.prevent="submit"
      >
        <el-form-item
          :label="$t('partner.kol.noteLabel')"
          required
        >
          <el-input
            v-model="note"
            name="kol-application-note"
            autocomplete="off"
            type="textarea"
            :rows="5"
            maxlength="500"
            show-word-limit
            :placeholder="$t('partner.kol.notePlaceholder')"
          />
        </el-form-item>
        <el-button
          type="primary"
          size="large"
          :loading="submitting"
          @click="submit"
        >
          {{ $t('partner.submitApplication') }}
        </el-button>
      </el-form>
    </el-card>

    <PayoutDialog
      v-model="payoutVisible"
      :title="$t('partner.payoutTitle', { role: $t('partner.kol.roleName') })"
      :submitting="submitting"
      @submit="submitPayout"
    />
  </section>
</template>

<script setup lang="ts">
import { errorMessage } from '@/utils/errors'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import PartnerSettlement from '@/components/partner/PartnerSettlement.vue'
import PartnerCardHeader from '@/components/partner/PartnerCardHeader.vue'
import PartnerStatGrid, { type PartnerStat } from '@/components/partner/PartnerStatGrid.vue'
import PayoutDialog from '@/components/partner/PayoutDialog.vue'
import {
  createKOLPayout,
  fetchKOLCommissions,
  fetchKOLPayouts,
  fetchKOLSession,
  fetchPartnerStates,
  submitApplication,
  type BalanceSummary,
  type KOLOverview,
  type PartnerApplication,
  type PartnerRoleStatus,
  type Payout,
  type PayoutMethod,
  type PartnerCommission,
} from '@/api/partner'
import { formatDateTime, formatMinorUnit } from '@/utils/format'
import { buildReferralLink } from '@/utils/share'
import {
  partnerCommissionStatus as commissionMeta,
  partnerPayoutStatus as payoutMeta,
} from '@/utils/partnerStatus'

const { t } = useI18n()
const loading = ref(true)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const roleStatus = ref<PartnerRoleStatus | ''>('')
const otherRoleAssigned = ref(false)
const application = ref<PartnerApplication | null>(null)
const note = ref('')
const overview = ref<KOLOverview | null>(null)
const commissions = ref<PartnerCommission[]>([])
const balances = ref<BalanceSummary[]>([])
const payouts = ref<Payout[]>([])

const referralLink = computed(() => {
  const code = overview.value?.partner.code
  return code ? buildReferralLink(code) : ''
})
const overviewStats = computed<PartnerStat[]>(() => [
  {
    key: 'attributed-users',
    label: t('partner.attributedUsers'),
    value: overview.value?.attributed_users ?? 0,
  },
  ...balances.value.map(balance => ({
    key: `balance-${balance.currency}`,
    label: t('partner.availableBalance', { currency: balance.currency }),
    value: formatMinorUnit(balance.total_minor, balance.currency),
  })),
])

async function load() {
  loading.value = true
  error.value = ''
  try {
    const states = await fetchPartnerStates()
    roleStatus.value = states.kol.status ?? ''
    otherRoleAssigned.value = !roleStatus.value && Boolean(states.scout.status)
    application.value = states.kol.application
    if (states.kol.status !== 'active') return

    const [session, commissionResult, payoutResult] = await Promise.all([
      fetchKOLSession(),
      fetchKOLCommissions(),
      fetchKOLPayouts(),
    ])
    overview.value = session.overview
    commissions.value = commissionResult.commissions
    balances.value = commissionResult.balances
    payouts.value = payoutResult.payouts
  } catch (err) {
    error.value = errorMessage(err) || t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!note.value.trim()) {
    ElMessage.warning(t('partner.kol.noteRequired'))
    return
  }
  submitting.value = true
  try {
    const result = await submitApplication('kol', note.value.trim())
    application.value = result.application
    ElMessage.success(t('partner.kol.applicationSubmitted'))
  } catch (err) {
    ElMessage.error(errorMessage(err) || t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

async function copy(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success(t('common.copied'))
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}

async function submitPayout(payload: { method: PayoutMethod; account: string }) {
  submitting.value = true
  try {
    const result = await createKOLPayout({ ...payload })
    payouts.value.unshift(result.payout)
    payoutVisible.value = false
    ElMessage.success(t('partner.payoutSubmitted'))
    await load()
  } catch (err) {
    ElMessage.error(errorMessage(err) || t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function commissionLabel(status: PartnerCommission['status']) {
  return commissionMeta(status).label
}

function commissionTagType(status: PartnerCommission['status']) {
  return commissionMeta(status).tag
}

function payoutLabel(status: Payout['status']) {
  return payoutMeta(status).label
}

function payoutTagType(status: Payout['status']) {
  return payoutMeta(status).tag
}

onMounted(load)
</script>

<style scoped>
.apply-card { max-width: 720px; }
.apply-card h2 { margin: 0 0 8px; font-size: 22px; font-weight: 700; }
.apply-card p { margin: 0 0 18px; color: var(--liki-text-2); }
.section-card { margin-bottom: 20px; }
.rules { margin-bottom: 14px; }
.overview-stats { margin-top: var(--space-4); }
</style>
