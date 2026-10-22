import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const fetchPartnerStates = vi.fn()
const submitApplication = vi.fn()

vi.mock('@/api/partner', () => ({
  fetchPartnerStates: (...args: unknown[]) => fetchPartnerStates(...args),
  submitApplication: (...args: unknown[]) => submitApplication(...args),
}))

vi.mock('@/composables/useMobile', () => ({
  useMobile: () => ({ value: false }),
}))

vi.mock('@/composables/useUnloadGuard', () => ({
  useUnloadGuard: () => {},
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

import { usePartnerRolePage } from '@/composables/usePartnerRolePage'

function setup(role: 'kol' | 'scout') {
  return usePartnerRolePage(role)
}

describe('usePartnerRolePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('loads role state and marks loaded', async () => {
    fetchPartnerStates.mockResolvedValue({
      kol: { status: 'active', application: null },
      scout: { status: '', application: null },
    })

    const page = setup('kol')
    await page.load()

    expect(page.loaded.value).toBe(true)
    expect(page.roleStatus.value).toBe('active')
    expect(page.otherRoleAssigned.value).toBe(false)
    expect(page.loading.value).toBe(false)
    expect(page.error.value).toBe('')
  })

  it('detects other role assigned when own role is empty', async () => {
    fetchPartnerStates.mockResolvedValue({
      kol: { status: '', application: null },
      scout: { status: 'active', application: null },
    })

    const page = setup('kol')
    await page.load()

    expect(page.otherRoleAssigned.value).toBe(true)
  })

  it('sets error on load failure', async () => {
    fetchPartnerStates.mockRejectedValue(new Error('network'))

    const page = setup('kol')
    await page.load()

    expect(page.error.value).toBeTruthy()
    expect(page.loaded.value).toBe(false)
  })

  it('calls fetchActive only when role is active', async () => {
    fetchPartnerStates.mockResolvedValue({
      kol: { status: 'pending', application: null },
      scout: { status: '', application: null },
    })

    const fetchActive = vi.fn()
    const page = setup('kol')
    await page.load(fetchActive)

    expect(fetchActive).not.toHaveBeenCalled()
  })

  it('skips fetchPartnerStates when skipRoleState is true', async () => {
    fetchPartnerStates.mockResolvedValue({
      kol: { status: 'active', application: null },
      scout: { status: '', application: null },
    })

    const page = setup('kol')
    await page.load() // initial load sets roleStatus to 'active'

    fetchPartnerStates.mockClear()
    const fetchActive = vi.fn()
    await page.load(fetchActive, { skipRoleState: true })

    expect(fetchPartnerStates).not.toHaveBeenCalled()
    expect(fetchActive).toHaveBeenCalledTimes(1)
  })

  it('submits application and clears form', async () => {
    fetchPartnerStates.mockResolvedValue({
      kol: { status: '', application: null },
      scout: { status: '', application: null },
    })
    submitApplication.mockResolvedValue({
      application: { status: 'pending', decision_reason: '' },
    })

    const page = setup('kol')
    page.form.note = '  please approve  '
    await page.submit()

    expect(submitApplication).toHaveBeenCalledWith('kol', 'please approve')
    expect(page.form.note).toBe('')
    expect(page.application.value?.status).toBe('pending')
  })

  it('formats commission status label and tag', () => {
    const page = setup('kol')
    expect(typeof page.commissionLabel('pending')).toBe('string')
    expect(typeof page.commissionTagType('pending')).toBe('string')
  })
})
