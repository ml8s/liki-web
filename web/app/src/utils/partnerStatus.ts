import { i18n } from '@/i18n'
import type { CommissionStatus, PayoutStatus } from '@/api/partner'

export type PartnerStatusTagType = 'success' | 'danger' | 'warning'

export interface PartnerStatusMeta {
  label: string
  tag: PartnerStatusTagType
}

export function partnerCommissionStatus(status: CommissionStatus): PartnerStatusMeta {
  const t = i18n.global.t
  const labels: Record<CommissionStatus, PartnerStatusMeta> = {
    pending: { label: t('partner.commissionStatus.pending'), tag: 'warning' },
    paid: { label: t('partner.commissionStatus.paid'), tag: 'success' },
    voided: { label: t('partner.commissionStatus.voided'), tag: 'danger' },
  }
  return labels[status]
}

export function partnerPayoutStatus(status: PayoutStatus): PartnerStatusMeta {
  const t = i18n.global.t
  const labels: Record<PayoutStatus, PartnerStatusMeta> = {
    requested: { label: t('partner.payoutStatus.requested'), tag: 'warning' },
    pending: { label: t('partner.payoutStatus.pending'), tag: 'warning' },
    approved: { label: t('partner.payoutStatus.approved'), tag: 'warning' },
    rejected: { label: t('partner.payoutStatus.rejected'), tag: 'danger' },
    paid: { label: t('partner.payoutStatus.paid'), tag: 'success' },
    failed: { label: t('partner.payoutStatus.failed'), tag: 'danger' },
  }
  return labels[status]
}
