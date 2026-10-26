/**
 * PII-free client failure reporting.
 *
 * Errors were previously swallowed into generic UI copy, which made protocol
 * and serialization regressions invisible. This reporter sends only a stable
 * code, the current path, locale and technical detail. It never sends birth
 * data, chat content, email addresses or provider payloads.
 */
import { postTelemetryJSON, telemetryDisabled } from './telemetry'

export interface ClientErrorEvent {
  code: string
  view?: string
  detail?: string
  transport?: string
  agent?: string
}

const reported = new Set<string>()

function sanitizeDetail(value: string | undefined): string {
  if (!value) return ''
  const cleaned = Array.from(value, char => {
    const code = char.codePointAt(0) ?? 0
    return code < 0x20 || code === 0x7f ? ' ' : char
  }).join('')
  return cleaned.trim().slice(0, 300)
}

export function reportClientError(event: ClientErrorEvent): void {
  if (telemetryDisabled()) return
  const code = event.code || 'unknown'
  if (reported.has(code)) return
  reported.add(code)

  const payload = {
    code,
    view: event.view ?? window.location.pathname,
    locale: document.documentElement.lang || navigator.language,
    transport: event.transport,
    agent: event.agent,
    detail: sanitizeDetail(event.detail),
  }
  postTelemetryJSON('/api/client-errors', payload)
}

/** Test helper: allow a fresh page-load dedupe state. */
export function resetClientErrorReporting(): void {
  reported.clear()
}
