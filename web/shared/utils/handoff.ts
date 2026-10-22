import { useFormDraftsStore } from '../stores/form-drafts'

const HANDOFF_TTL_MS = 30 * 60 * 1000

/** Semantic keys for cross-surface handoff data. */
export const HANDOFF_KEYS = {
  birth: 'handoff:birth',
} as const

interface HandoffEnvelope<T> {
  data: T
  createdAt: number
}

/** Save data with a timestamp for cross-surface handoff. */
export function saveHandoff(key: string, data: unknown): void {
  const store = useFormDraftsStore()
  const envelope: HandoffEnvelope<unknown> = {
    data,
    createdAt: Date.now(),
  }
  store.set(key, envelope)
}

/** Read once, validate TTL, then consume (one-time). */
export function consumeHandoff<T>(key: string, maxAgeMs: number = HANDOFF_TTL_MS): T | null {
  const store = useFormDraftsStore()
  const envelope = store.consume<HandoffEnvelope<T>>(key)
  if (!envelope) return null
  if (typeof envelope.createdAt !== 'number') return null
  if (Date.now() - envelope.createdAt > maxAgeMs) return null
  return envelope.data
}
