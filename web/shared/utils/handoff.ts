import { useFormDraftsStore } from '../stores/form-drafts'
import type { ConsultationContext, ConsultationIntent } from '../domain/consultation'

const HANDOFF_TTL_MS = 30 * 60 * 1000

/** Semantic keys for cross-surface handoff data. */
export const HANDOFF_KEYS = {
  birth: 'handoff:birth',
  consultation: 'handoff:consultation',
} as const

interface HandoffEnvelope<T> {
  data: T
  createdAt: number
}

export function saveConsultationHandoff(context: unknown): void {
	saveHandoff(HANDOFF_KEYS.consultation, context)
}

export function peekConsultationHandoff<T extends ConsultationContext>(
	intent: ConsultationIntent,
): T | null {
	const store = useFormDraftsStore()
	const envelope = store.get<HandoffEnvelope<T>>(HANDOFF_KEYS.consultation)
	if (!envelope || typeof envelope.createdAt !== 'number') return null
	if (Date.now() - envelope.createdAt > HANDOFF_TTL_MS) return null
	if (!envelope.data || envelope.data.intent !== intent) return null
	return envelope.data
}

export function consumeConsultationHandoff<T extends ConsultationContext>(
	intent: ConsultationIntent,
): T | null {
	const store = useFormDraftsStore()
	const envelope = store.consume<HandoffEnvelope<T>>(HANDOFF_KEYS.consultation)
	if (!envelope || typeof envelope.createdAt !== 'number') return null
	if (Date.now() - envelope.createdAt > HANDOFF_TTL_MS) return null
	if (!envelope.data || envelope.data.intent !== intent) return null
	return envelope.data
}

export function clearConsultationHandoff(): void {
	useFormDraftsStore().clear(HANDOFF_KEYS.consultation)
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
