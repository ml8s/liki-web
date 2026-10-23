import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDraftsStore } from '@shared/stores/form-drafts'
import type { ConsultationContext } from '@shared/domain/consultation'
import {
  consumeConsultationHandoff,
  consumeHandoff,
  saveConsultationHandoff,
  saveHandoff,
  HANDOFF_KEYS,
} from '@shared/utils/handoff'

describe('handoff', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
    useFormDraftsStore().drafts = {}
  })

  it('saves and consumes handoff data round-trip', () => {
    saveHandoff(HANDOFF_KEYS.birth, { birth_date: '2000-01-01', gender: 'female' })

    const result = consumeHandoff<{ birth_date: string; gender: string }>(HANDOFF_KEYS.birth)
    expect(result).toEqual({ birth_date: '2000-01-01', gender: 'female' })
  })

  it('consumed handoff is one-time (second read returns null)', () => {
    saveHandoff(HANDOFF_KEYS.birth, { test: true })
    consumeHandoff(HANDOFF_KEYS.birth)

    expect(consumeHandoff(HANDOFF_KEYS.birth)).toBeNull()
  })

  it('rejects expired handoff', () => {
    saveHandoff(HANDOFF_KEYS.birth, { test: true })
    // Expire immediately.
    expect(consumeHandoff(HANDOFF_KEYS.birth, -1)).toBeNull()
  })

  it('returns null for missing or malformed data', () => {
    expect(consumeHandoff('nonexistent-key')).toBeNull()

    // Write garbage (no envelope) directly.
    useFormDraftsStore().set('bad', { foo: 'bar' })
    expect(consumeHandoff('bad')).toBeNull()
  })

  it('consumes a matching consultation handoff exactly once', () => {
    saveConsultationHandoff({ version: 1, scenario: 'divination', question: '項目能否成交' })

    const first = consumeConsultationHandoff<ConsultationContext>('divination')
    expect(first).toMatchObject({ scenario: 'divination', question: '項目能否成交' })
    expect(consumeConsultationHandoff('divination')).toBeNull()
  })
})
