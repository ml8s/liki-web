import { defineStore } from 'pinia'
import { useFormDraftsStore } from '@shared/stores/form-drafts'
import { intentGroupById, type IntentGroupId } from '@shared/domain/intent-flow'

const DEFAULT_INTENT_GROUP: IntentGroupId = 'natal'

function draftKey(personId: string | null, groupId: IntentGroupId): string {
  return `flow:${personId ?? 'none'}:${groupId}`
}

interface IntentFlowState {
  personId: string | null
  groupId: IntentGroupId
}

/**
 * P1 selection state for the four-domain consultation shell.
 * Session ownership stays with the existing chat API until Phase 2.
 */
export const useIntentFlowStore = defineStore('intent-flow', {
  state: (): IntentFlowState => ({
    personId: null,
    groupId: DEFAULT_INTENT_GROUP,
  }),
  getters: {
    currentIntentGroup: state => intentGroupById(state.groupId),
    draftKey: state => draftKey(state.personId, state.groupId),
  },
  actions: {
    setPerson(personId: string | null) {
      this.personId = personId
    },
    setGroup(groupId: IntentGroupId) {
      if (!intentGroupById(groupId)) throw new TypeError(`Unknown intent group: ${groupId}`)
      this.groupId = groupId
    },
    getDraft(): string {
      return useFormDraftsStore().get<string>(draftKey(this.personId, this.groupId)) ?? ''
    },
    setDraft(value: string) {
      const normalized = value ?? ''
      if (normalized) useFormDraftsStore().set(draftKey(this.personId, this.groupId), normalized)
      else useFormDraftsStore().clear(draftKey(this.personId, this.groupId))
    },
    clearDraft() {
      this.setDraft('')
    },
  },
})
