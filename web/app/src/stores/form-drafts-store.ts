import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'

const FORM_DRAFTS_STORAGE_KEY = 'liki:form-drafts:v1'

/**
 * One store owns all short-lived form drafts. Values stay in sessionStorage so a
 * refresh restores work, while closing the tab removes sensitive form data.
 */
export const useFormDraftsStore = defineStore('form-drafts', () => {
  const drafts = useStorage<Record<string, unknown>>(
    FORM_DRAFTS_STORAGE_KEY,
    {},
    sessionStorage,
    { flush: 'sync' },
  )

  function get<T extends object>(key: string): Partial<T> | null {
    const value = drafts.value[key]
    return value && typeof value === 'object' ? value as Partial<T> : null
  }

  function set<T extends object>(key: string, value: T): void {
    drafts.value[key] = value
  }

  function clear(key: string): void {
    delete drafts.value[key]
  }

  return {
    drafts,
    get,
    set,
    clear,
  }
})
