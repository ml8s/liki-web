import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFormDraftsStore } from '@/stores/form-drafts-store'

interface DemoDraft {
  name: string
}

describe('form drafts store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
    useFormDraftsStore().clearAll()
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

    const raw = sessionStorage.getItem('liki:form-drafts:v1')
    expect(raw).toContain('birth-profile:new:user-1')
    expect(raw).toContain('Alice')
  })
})
