import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useIntentFlowStore } from '@/stores/intent-flow'
import { useFormDraftsStore } from '@shared/stores/form-drafts'

describe('group flow store', () => {
  beforeEach(() => {
    sessionStorage.clear()
    setActivePinia(createPinia())
    useFormDraftsStore().drafts = {}
  })

  it('starts with no person in the default consultation group', () => {
    const flow = useIntentFlowStore()

    expect(flow.personId).toBeNull()
    expect(flow.groupId).toBe('natal')
    expect(flow.currentIntentGroup?.id).toBe('natal')
    expect(flow.draftKey).toBe('flow:none:natal')
  })

  it('keys drafts by person and group so contexts never share text', () => {
    const flow = useIntentFlowStore()
    flow.setPerson('person-a')
    flow.setGroup('natal')
    flow.setDraft('natal question')

    flow.setGroup('divination')
    expect(flow.getDraft()).toBe('')

    flow.setGroup('natal')
    expect(flow.getDraft()).toBe('natal question')
  })

  it('switching person switches the draft view without mutating the source', () => {
    const flow = useIntentFlowStore()
    flow.setPerson('person-a')
    flow.setDraft('张三问题')

    flow.setPerson('person-b')
    expect(flow.getDraft()).toBe('')

    flow.setPerson('person-a')
    expect(flow.getDraft()).toBe('张三问题')
  })

  it('keeps drafts in the form-draft store using the explicit context key', () => {
    const flow = useIntentFlowStore()
    flow.setPerson('person-a')
    flow.setGroup('fengshui')
    flow.setDraft('办公室朝向')

    expect(flow.draftKey).toBe('flow:person-a:fengshui')
    expect(useFormDraftsStore().drafts[flow.draftKey]).toBe('办公室朝向')
  })

  it('clears only the visible draft', () => {
    const flow = useIntentFlowStore()
    flow.setPerson('person-a')
    flow.setDraft('visible draft')
    flow.setGroup('divination')
    flow.setDraft('divination draft')

    flow.clearDraft()

    flow.setGroup('natal')
    expect(flow.getDraft()).toBe('visible draft')
  })

  it('removes an empty draft key instead of retaining an empty value', () => {
    const flow = useIntentFlowStore()
    flow.setDraft('temporary')
    flow.clearDraft()

    expect(Object.hasOwn(useFormDraftsStore().drafts, flow.draftKey)).toBe(false)
  })

  it('rejects unknown groups', () => {
    const flow = useIntentFlowStore()
    expect(() => flow.setGroup('unknown' as never)).toThrow(TypeError)
  })
})
