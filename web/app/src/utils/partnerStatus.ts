import { i18n } from '@/i18n'
import type { CommissionStatus, PartnerProgramState, PartnerRole, PayoutStatus } from '@/api/partner'

export type PartnerStatusTagType = 'success' | 'danger' | 'warning' | 'info'

export interface PartnerStatusMeta {
  label: string
  tag: PartnerStatusTagType
}

/** 角色名统一走 i18n，避免页面内硬编码 KOL / Scout。 */
export function partnerRoleName(role: PartnerRole): string {
  return i18n.global.t(`partner.${role}.roleName`)
}

/** 合作角色状态（含申请进度与角色冲突）统一映射，供 Partner 各页共用。 */
export function partnerRoleState(state: PartnerProgramState, otherRoleAssigned = false): PartnerStatusMeta {
  const t = i18n.global.t
  if (state.status === 'active') return { label: t('partner.state.active'), tag: 'success' }
  if (state.status === 'retired') return { label: t('partner.state.retired'), tag: 'info' }
  if (state.status === 'frozen') return { label: t('partner.state.frozen'), tag: 'danger' }
  if (state.application?.status === 'pending') return { label: t('partner.state.pending'), tag: 'warning' }
  if (state.application?.status === 'rejected') return { label: t('partner.state.reapply'), tag: 'warning' }
  if (otherRoleAssigned) return { label: t('partner.state.conflict'), tag: 'info' }
  return { label: t('partner.state.available'), tag: 'info' }
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
