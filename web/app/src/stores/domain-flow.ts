import { defineStore } from 'pinia'
import { useFormDraftsStore } from '@shared/stores/form-drafts'
import { appDomainById, type AppDomainId } from '@shared/domain/flow'

const DEFAULT_DOMAIN: AppDomainId = 'natal'

function draftKey(personId: string | null, domainId: AppDomainId): string {
  return `flow:${personId ?? 'none'}:${domainId}`
}

interface DomainFlowState {
  personId: string | null
  domainId: AppDomainId
}

/**
 * P1 selection state for the four-domain consultation shell.
 * Session ownership stays with the existing chat API until Phase 2.
 */
export const useDomainFlowStore = defineStore('domain-flow', {
  state: (): DomainFlowState => ({
    personId: null,
    domainId: DEFAULT_DOMAIN,
  }),
  getters: {
    currentDomain: state => appDomainById(state.domainId),
    draftKey: state => draftKey(state.personId, state.domainId),
  },
  actions: {
    setPerson(personId: string | null) {
      this.personId = personId
    },
    setDomain(domainId: AppDomainId) {
      if (!appDomainById(domainId)) throw new TypeError(`Unknown app domain: ${domainId}`)
      this.domainId = domainId
    },
    getDraft(): string {
      return useFormDraftsStore().get<string>(draftKey(this.personId, this.domainId)) ?? ''
    },
    setDraft(value: string) {
      const normalized = value ?? ''
      if (normalized) useFormDraftsStore().set(draftKey(this.personId, this.domainId), normalized)
      else useFormDraftsStore().clear(draftKey(this.personId, this.domainId))
    },
    clearDraft() {
      this.setDraft('')
    },
  },
})
