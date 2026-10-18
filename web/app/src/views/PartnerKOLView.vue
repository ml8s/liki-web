<template>
  <section>
    <AppPageHeader
      :title="$t('partner.kol.title')"
      :subtitle="$t('partner.kol.subtitle')"
    >
      <template #actions>
        <el-button
          v-if="showRefresh"
          :loading="loading"
          @click="load"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-skeleton
      v-if="loading && !loaded"
      :rows="6"
      animated
      aria-busy="true"
    />
    <el-result
      v-else-if="error && !loaded"
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
        class="load-alert"
      />

      <div
        v-loading="loading"
        :aria-busy="loading"
      >
        <template v-if="roleStatus === 'active'">
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
                @click="payoutVisible = true"
              >
                {{ $t('partner.settlement.requestPayout') }}
              </el-button>
            </el-empty>
          </el-card>
        </template>

        <PartnerSettlement
          v-else-if="roleStatus === 'retired'"
          role="kol"
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
          :title="$t('partner.otherRoleTitle', { role: $t('partner.scout.roleName') })"
          :description="$t('partner.roleConflictDescription')"
        />

        <el-result
          v-else-if="application?.status === 'pending'"
          icon="info"
          :title="$t('partner.pendingTitle', { role: roleName })"
          :description="$t('partner.kol.pendingDescription')"
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
          <h2>{{ $t('partner.kol.applyTitle') }}</h2>
          <p>{{ $t('partner.kol.applyDescription') }}</p>
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            @submit.prevent="submit"
          >
            <el-form-item
              :label="$t('partner.kol.noteLabel')"
              prop="note"
              required
            >
              <el-input
                v-model="form.note"
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
              native-type="submit"
              :loading="submitting"
            >
              {{ $t('partner.submitApplication') }}
            </el-button>
          </el-form>
        </el-card>
      </div>
    </template>

    <PayoutDialog
      v-model="payoutVisible"
      :title="$t('partner.payoutTitle', { role: roleName })"
      :submitting="submitting"
      @submit="submitPayout"
    />
  </section>
</template>

<script setup lang="ts">
import { errorMessage } from '@/utils/errors'
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FormInstance, FormRules } from 'element-plus'
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
import { formatDateTime, formatMinorUnit, productName } from '@/utils/format'
import { buildReferralLink } from '@/utils/share'
import { useMobile } from '@/composables/useMobile'
import { useUnloadGuard } from '@/composables/useUnloadGuard'
import {
  partnerCommissionStatus as commissionMeta,
  partnerPayoutStatus as payoutMeta,
  partnerRoleName,
} from '@/utils/partnerStatus'

const { t } = useI18n()
const loading = ref(true)
const loaded = ref(false)
const submitting = ref(false)
const payoutVisible = ref(false)
const error = ref('')
const roleStatus = ref<PartnerRoleStatus | ''>('')
const otherRoleAssigned = ref(false)
const application = ref<PartnerApplication | null>(null)
const overview = ref<KOLOverview | null>(null)
const commissions = ref<PartnerCommission[]>([])
const balances = ref<BalanceSummary[]>([])
const payouts = ref<Payout[]>([])
const formRef = ref<FormInstance>()
const form = reactive({ note: '' })
const isMobile = useMobile(480)

const roleName = computed(() => partnerRoleName('kol'))
const showRefresh = computed(() => loading.value || Boolean(error.value) || roleStatus.value === 'active')
const rules = computed<FormRules>(() => ({
  note: [{ required: true, whitespace: true, message: t('partner.kol.noteRequired'), trigger: 'blur' }],
}))
const dirty = computed(() => form.note.trim().length > 0 && application.value?.status !== 'pending')
useUnloadGuard(dirty)

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
    loaded.value = true
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
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    const result = await submitApplication('kol', form.note.trim())
    application.value = result.application
    form.note = ''
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

function rejectionTitle(reason?: string) {
  return reason ? t('partner.rejectedTitle', { reason }) : t('partner.rejectedTitleNoReason')
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
.apply-card h2 { margin: 0 0 var(--space-2); font-size: var(--font-size-title-md); font-weight: 700; }
.apply-card p { margin: 0 0 var(--space-4); color: var(--liki-text-2); }
.section-card { margin-bottom: var(--space-5); }
.rules, .load-alert { margin-bottom: var(--space-4); }
.overview-stats { margin-top: var(--space-4); }
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
