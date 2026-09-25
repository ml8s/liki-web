<template>
  <section>
    <AppPageHeader
      :title="$t('partner.kol.title')"
      :subtitle="$t('partner.kol.subtitle')"
    >
      <template #actions>
        <el-button :loading="loading" @click="load">{{ $t('common.refresh') }}</el-button>
      </template>
    </AppPageHeader>

    <el-skeleton v-if="loading" :rows="6" animated aria-busy="true" />
    <el-alert v-else-if="error" type="error" :title="error" show-icon :closable="false" />

    <template v-else-if="roleStatus === 'active'">
      <el-card class="section-card" shadow="never">
        <template #header>
          <div class="card-head">
            <span>{{ $t('partner.referralLink') }}</span>
            <el-button size="small" @click="copy(referralLink)">{{ $t('common.copy') }}</el-button>
          </div>
        </template>
        <el-input :model-value="referralLink" readonly />
        <div class="stat-grid">
          <div class="stat">
            <span class="label">{{ $t('partner.attributedUsers') }}</span>
            <strong>{{ overview?.attributed_users ?? 0 }}</strong>
          </div>
          <div v-for="balance in balances" :key="balance.currency" class="stat">
            <span class="label">{{ $t('partner.availableBalance', { currency: balance.currency }) }}</span>
            <strong>{{ formatMinorUnit(balance.total_minor, balance.currency) }}</strong>
          </div>
        </div>
      </el-card>

      <el-card class="section-card" shadow="never">
        <template #header><span>{{ $t('partner.commissions') }}</span></template>
        <el-table :data="commissions" :empty-text="$t('partner.noCommissions')">
          <el-table-column prop="product" :label="$t('partner.product')" min-width="130" />
          <el-table-column :label="$t('partner.orderAmount')" width="130">
            <template #default="{ row }">{{ formatMinorUnit(row.amount_minor, row.currency) }}</template>
          </el-table-column>
          <el-table-column :label="$t('partner.commission')" width="130">
            <template #default="{ row }">{{ formatMinorUnit(row.referrer_commission_minor, row.currency) }}</template>
          </el-table-column>
          <el-table-column :label="$t('partner.status')" width="110">
            <template #default="{ row }">
              <el-tag size="small" :type="commissionTagType(row.status)">
                {{ commissionLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('partner.time')" width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card class="section-card" shadow="never">
        <template #header>
          <div class="card-head">
            <span>{{ $t('partner.payouts') }}</span>
            <el-button type="primary" size="small" @click="payoutVisible = true">{{ $t('partner.settlement.requestPayout') }}</el-button>
          </div>
        </template>
        <el-alert
          :title="$t('partner.payoutRules')"
          type="info"
          :closable="false"
          show-icon
          class="rules"
        />
        <el-table :data="payouts" :empty-text="$t('partner.noPayouts')">
          <el-table-column :label="$t('partner.amount')" width="130">
            <template #default="{ row }">{{ formatMinorUnit(row.amount_minor, row.currency) }}</template>
          </el-table-column>
          <el-table-column prop="method" :label="$t('partner.method')" width="120" />
          <el-table-column :label="$t('partner.account')" min-width="180">
            <template #default="{ row }">{{ row.account }}</template>
          </el-table-column>
          <el-table-column :label="$t('partner.status')" width="120">
            <template #default="{ row }">
              <el-tag size="small" :type="payoutTagType(row.status)">{{ payoutLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('partner.requestedAt')" width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>

    <PartnerSettlement v-else-if="roleStatus === 'retired'" role="kol" />

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

    <el-card v-else class="apply-card" shadow="never">
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
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item :label="$t('partner.kol.noteLabel')" required>
          <el-input
            v-model="note"
            type="textarea"
            :rows="5"
            maxlength="500"
            show-word-limit
            :placeholder="$t('partner.kol.notePlaceholder')"
          />
        </el-form-item>
        <el-button type="primary" size="large" :loading="submitting" @click="submit">{{ $t('partner.submitApplication') }}</el-button>
      </el-form>
    </el-card>

    <el-dialog v-model="payoutVisible" :title="$t('partner.payoutTitle', { role: $t('partner.kol.roleName') })" width="420px">
      <el-form label-position="top">
        <el-form-item :label="$t('partner.method')">
          <el-radio-group v-model="payoutForm.method">
            <el-radio-button value="alipay">{{ $t('partner.alipay') }}</el-radio-button>
            <el-radio-button value="paypal">PayPal</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('partner.account')" required>
          <el-input v-model="payoutForm.account" autocomplete="off" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="payoutVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitPayout">{{ $t('common.submit') }}</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import AppPageHeader from '@/components/app/AppPageHeader.vue'
import PartnerSettlement from '@/components/partner/PartnerSettlement.vue'
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
  type ReferralCommission,
} from '@/api/referral'
import { formatDateTime, formatMinorUnit } from '@/utils/format'

const { t } = useI18n()
const loading = ref(true)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const roleStatus = ref<PartnerRoleStatus | ''>('')
const otherRoleAssigned = ref(false)
const application = ref<PartnerApplication | null>(null)
const canSubmit = ref(false)
const note = ref('')
const overview = ref<KOLOverview | null>(null)
const commissions = ref<ReferralCommission[]>([])
const balances = ref<BalanceSummary[]>([])
const payouts = ref<Payout[]>([])
const payoutForm = reactive({ method: 'alipay' as PayoutMethod, account: '' })

const referralLink = computed(() => {
  const code = overview.value?.referrer.code
  return code ? `${window.location.origin}/r/${code}` : ''
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const states = await fetchPartnerStates()
    roleStatus.value = states.kol.status ?? ''
    otherRoleAssigned.value = !roleStatus.value && Boolean(states.scout.status)
    application.value = states.kol.application
    canSubmit.value = states.kol.can_submit
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
    error.value = err instanceof Error ? err.message : t('common.loadFailed')
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
    canSubmit.value = false
    ElMessage.success(t('partner.kol.applicationSubmitted'))
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t('partner.submitFailed'))
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

async function submitPayout() {
  if (!payoutForm.account.trim()) {
    ElMessage.warning(t('partner.accountRequired'))
    return
  }
  submitting.value = true
  try {
    const result = await createKOLPayout({ ...payoutForm, account: payoutForm.account.trim() })
    payouts.value.unshift(result.payout)
    payoutVisible.value = false
    ElMessage.success(t('partner.payoutSubmitted'))
    await load()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t('partner.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function commissionLabel(status: ReferralCommission['status']) {
  const labels = { pending: t('partner.commissionStatus.pending'), paid: t('partner.commissionStatus.paid'), voided: t('partner.commissionStatus.voided') }
  return labels[status]
}

function commissionTagType(status: ReferralCommission['status']) {
  if (status === 'paid') return 'success'
  if (status === 'voided') return 'danger'
  return 'warning'
}

function payoutLabel(status: Payout['status']) {
  const labels: Record<Payout['status'], string> = {
    requested: t('partner.payoutStatus.requested'), pending: t('partner.payoutStatus.pending'), approved: t('partner.payoutStatus.approved'),
    rejected: t('partner.payoutStatus.rejected'), paid: t('partner.payoutStatus.paid'), failed: t('partner.payoutStatus.failed'),
  }
  return labels[status]
}

function payoutTagType(status: Payout['status']) {
  if (status === 'paid') return 'success'
  if (status === 'rejected' || status === 'failed') return 'danger'
  return 'warning'
}

onMounted(load)
</script>

<style scoped>
.apply-card { max-width: 720px; }
.apply-card h2 { margin: 0 0 8px; font-size: 22px; font-weight: 700; }
.apply-card p { margin: 0 0 18px; color: var(--liki-text-2); }
.section-card { margin-bottom: 20px; }
.card-head { display: flex; align-items: center; justify-content: space-between; font-weight: 600; }
.rules { margin-bottom: 14px; }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; margin-top: 18px; }
.stat { border: 1px solid var(--liki-line); border-radius: 12px; padding: 14px; background: var(--liki-paper); }
.stat .label { display: block; margin-bottom: 6px; color: var(--liki-text-2); font-size: 13px; }
.stat strong { font-size: 20px; }
</style>
