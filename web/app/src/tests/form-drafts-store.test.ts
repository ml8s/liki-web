import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

interface DemoDraft {
  name: string
}

describe('form drafts store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
    useFormDraftsStore().drafts = {}
  })

  it('sets, gets, and clears a namespaced draft', () => {
    const store = useFormDraftsStore()

    store.set<DemoDraft>('birth-profile:new:user-1', { name: 'Alice' })
    expect(store.get<DemoDraft>('birth-profile:new:user-1')).toEqual({ name: 'Alice' })

    store.clear('birth-profile:new:user-1')
    expect(store.get<DemoDraft>('birth-profile:new:user-1')).toBeNull()
  })

  it('persists drafts in sessionStorage under one storage key', () => {
    const store = useFormDraftsStore()
    store.set<DemoDraft>('birth-profile:new:user-1', { name: 'Alice' })

    const raw = sessionStorage.getItem('liki:form-drafts')
    expect(raw).toContain('birth-profile:new:user-1')
    expect(raw).toContain('Alice')
  })

  it('consume reads once and removes', () => {
    const store = useFormDraftsStore()
    store.set('handoff:birth', { birth_date: '2000-01-01' })

    const first = store.consume<{ birth_date: string }>('handoff:birth')
    expect(first).toEqual({ birth_date: '2000-01-01' })
    expect(store.get('handoff:birth')).toBeNull()
  })

  it('silently handles unavailable sessionStorage', () => {
    const original = sessionStorage.setItem
    sessionStorage.setItem = () => { throw new Error('QuotaExceeded') }
    const store = useFormDraftsStore()
    expect(() => store.set('test', { foo: 'bar' })).not.toThrow()
    sessionStorage.setItem = original
  })
})
