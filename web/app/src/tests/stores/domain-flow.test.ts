import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDomainFlowStore } from '@/stores/domain-flow'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

describe('domain flow store', () => {
  beforeEach(() => {
    sessionStorage.clear()
    setActivePinia(createPinia())
    useFormDraftsStore().drafts = {}
  })

  it('starts with no person in the default consultation domain', () => {
    const flow = useDomainFlowStore()

    expect(flow.personId).toBeNull()
    expect(flow.domainId).toBe('natal')
    expect(flow.currentDomain?.id).toBe('natal')
    expect(flow.draftKey).toBe('flow:none:natal')
  })

  it('keys drafts by person and domain so contexts never share text', () => {
    const flow = useDomainFlowStore()
    flow.setPerson('person-a')
    flow.setDomain('natal')
    flow.setDraft('natal question')

    flow.setDomain('divination')
    expect(flow.getDraft()).toBe('')

    flow.setDomain('natal')
    expect(flow.getDraft()).toBe('natal question')
  })

  it('switching person switches the draft view without mutating the source', () => {
    const flow = useDomainFlowStore()
    flow.setPerson('person-a')
    flow.setDraft('张三问题')

    flow.setPerson('person-b')
    expect(flow.getDraft()).toBe('')

    flow.setPerson('person-a')
    expect(flow.getDraft()).toBe('张三问题')
  })

  it('keeps drafts in the form-draft store using the explicit context key', () => {
    const flow = useDomainFlowStore()
    flow.setPerson('person-a')
    flow.setDomain('fengshui')
    flow.setDraft('办公室朝向')

    expect(flow.draftKey).toBe('flow:person-a:fengshui')
    expect(useFormDraftsStore().drafts[flow.draftKey]).toBe('办公室朝向')
  })

  it('clears only the visible draft', () => {
    const flow = useDomainFlowStore()
    flow.setPerson('person-a')
    flow.setDraft('visible draft')
    flow.setDomain('divination')
    flow.setDraft('divination draft')

    flow.clearDraft()

    flow.setDomain('natal')
    expect(flow.getDraft()).toBe('visible draft')
  })

  it('removes an empty draft key instead of retaining an empty value', () => {
    const flow = useDomainFlowStore()
    flow.setDraft('temporary')
    flow.clearDraft()

    expect(Object.hasOwn(useFormDraftsStore().drafts, flow.draftKey)).toBe(false)
  })

  it('rejects unknown domains', () => {
    const flow = useDomainFlowStore()
    expect(() => flow.setDomain('unknown' as never)).toThrow(TypeError)
  })
})
