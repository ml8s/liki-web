import { defineStore } from 'pinia'
import { ref } from 'vue'

const FORM_DRAFTS_STORAGE_KEY = 'liki:form-drafts'

function readFromStorage(): Record<string, unknown> {
  try {
    const raw = sessionStorage.getItem(FORM_DRAFTS_STORAGE_KEY)
    return raw ? (JSON.parse(raw) as Record<string, unknown>) : {}
  } catch {
    return {}
  }
}

function writeToStorage(drafts: Record<string, unknown>): void {
  try {
    sessionStorage.setItem(FORM_DRAFTS_STORAGE_KEY, JSON.stringify(drafts))
  } catch {
    // sessionStorage unavailable (private mode, quota exceeded): silent no-op.
  }
}

/**
 * Unified cross-surface form draft store (marketing islands + App).
 * Values persist in sessionStorage for the tab lifetime only.
 * All storage access is guarded: private mode / quota errors are silently ignored.
 */
export const useFormDraftsStore = defineStore('form-drafts', () => {
  const drafts = ref<Record<string, unknown>>(readFromStorage())

  function persist(): void {
    writeToStorage(drafts.value)
  }

  function get<T>(key: string): T | null {
    const value = drafts.value[key]
    return value !== undefined ? (value as T) : null
  }

  function set<T>(key: string, value: T): void {
    drafts.value[key] = value
    persist()
  }

  function clear(key: string): void {
    delete drafts.value[key]
    persist()
  }

  /** Read once and immediately remove (one-time handoff semantics). */
  function consume<T>(key: string): T | null {
    const value = get<T>(key)
    if (value !== null) clear(key)
    return value
  }

  return { drafts, get, set, clear, consume }
})
