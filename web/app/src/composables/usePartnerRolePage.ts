import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage } from '@/utils/errors'
import { useMobile } from '@/composables/useMobile'
import { useUnloadGuard } from '@/composables/useUnloadGuard'
import {
  fetchPartnerStates,
  submitApplication,
  type PartnerApplication,
  type PartnerCommission,
  type PartnerRoleStatus,
  type Payout,
} from '@/api/partner'
import {
  partnerCommissionStatus,
  partnerRoleName,
} from '@/utils/partnerStatus'

export type PartnerRole = 'kol' | 'scout'

/**
 * Shared lifecycle for PartnerKOLView / PartnerScoutView.
 * Owns role state, loading/error, application form, and formatting helpers.
 * Role-specific data fetching is done in the view's `loadActive` callback.
 */
export function usePartnerRolePage(role: PartnerRole) {
  const { t } = useI18n()
  const loading = ref(true)
  const loaded = ref(false)
  const submitting = ref(false)
  const error = ref('')
  const roleStatus = ref<PartnerRoleStatus | ''>('')
  const otherRoleAssigned = ref(false)
  const application = ref<PartnerApplication | null>(null)
  const payouts = ref<Payout[]>([])
  const form = reactive({ note: '' })
  const isMobile = useMobile(480)

  const roleName = partnerRoleName(role)
  const showRefresh = computed(() => loading.value || Boolean(error.value) || roleStatus.value === 'active')
  const dirty = computed(() => form.note.trim().length > 0 && application.value?.status !== 'pending')
  useUnloadGuard(dirty)

  /** Load role state + active data. Pass skipRoleState when only refreshing active data (e.g. after payout). */
  async function load(fetchActive?: () => Promise<void>, options?: { skipRoleState?: boolean }) {
    loading.value = true
    error.value = ''
    try {
      if (!options?.skipRoleState) {
        const states = await fetchPartnerStates()
        loaded.value = true
        roleStatus.value = states[role].status ?? ''
        otherRoleAssigned.value = !roleStatus.value && Boolean(states[role === 'kol' ? 'scout' : 'kol'].status)
        application.value = states[role].application
      }
      if (roleStatus.value === 'active' && fetchActive) await fetchActive()
    } catch (err) {
      error.value = errorMessage(err) || t('common.loadFailed')
    } finally {
      loading.value = false
    }
  }

  async function submit() {
    submitting.value = true
    try {
      const result = await submitApplication(role, form.note.trim())
      application.value = result.application
      form.note = ''
      ElMessage.success(t(`partner.${role}.applicationSubmitted`))
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

  function commissionLabel(status: PartnerCommission['status']) {
    return partnerCommissionStatus(status).label
  }

  function commissionTagType(status: PartnerCommission['status']) {
    return partnerCommissionStatus(status).tag
  }

  return {
    loading, loaded, submitting, error,
    roleStatus, otherRoleAssigned, application,
    payouts, form, isMobile,
    roleName, showRefresh,
    load, submit, copy,
    commissionLabel, commissionTagType,
  }
}
